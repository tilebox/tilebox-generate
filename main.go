// Package tilebox-generate is a CLI tool to generate Tilebox datasets types for Go.
//
// Usage: tilebox-generate --dataset open_data.copernicus.sentinel1_sar
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protoplugin"
	"github.com/tilebox/structconf"
	tileboxdatasets "github.com/tilebox/tilebox-go/datasets/v1"
	datasetsv1 "github.com/tilebox/tilebox-go/protogen/go/datasets/v1"
	gengo "google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/pluginpb"
)

type Config struct {
	TileboxAPIKey string `validate:"required" help:"A Tilebox API key"`
	Dataset       string `env:"-"             validate:"required"              help:"A valid dataset slug e.g. 'open_data.copernicus.sentinel1_sar'"`
	Out           string `env:"-"             default:"protogen"               help:"A directory to write the output to"`
	Package       string `env:"-"             default:"tilebox.v1"             help:"Package name"`
	Name          string `env:"-"             help:"Override the message name"`
}

func pointer[T any](x T) *T {
	return &x
}

func main() {
	ctx := context.Background()

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "--help")
	}
	cfg := &Config{}
	structconf.MustLoadAndValidate(cfg,
		"tilebox-generate",
		structconf.WithDescription("Generate Tilebox datasets types for Go"),
	)

	client := tileboxdatasets.NewClient(tileboxdatasets.WithAPIKey(cfg.TileboxAPIKey))
	dataset, err := client.Dataset(ctx, cfg.Dataset)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get dataset", slog.Any("error", err))
		return
	}

	datasetFileDescriptor := dataset.Type.GetDescriptorSet().GetFile()[0]

	if cfg.Package != "" {
		datasetFileDescriptor.Package = &cfg.Package
		datasetFileDescriptor.Options = &descriptorpb.FileOptions{
			GoPackage: pointer(strings.ReplaceAll(datasetFileDescriptor.GetPackage(), ".", "/")),
		}
	}
	if cfg.Name != "" {
		datasetFileDescriptor.GetMessageType()[0].Name = &cfg.Name
	}
	datasetFileDescriptor.Name = pointer(path.Join(
		strings.ReplaceAll(datasetFileDescriptor.GetPackage(), ".", "/"),
		fmt.Sprintf("%s.proto", datasetFileDescriptor.GetMessageType()[0].GetName()),
	))

	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{datasetFileDescriptor.GetName()},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto),
			protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
			protodesc.ToFileDescriptorProto(datasetsv1.File_datasets_v1_well_known_types_proto),
			datasetFileDescriptor,
		},
	}
	response, err := generateCode(request)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate code", slog.Any("error", err))
		return
	}

	file := response.GetFile()[0] // we only generate one file
	outputPath := filepath.Join(cfg.Out, file.GetName())
	err = writeToDisk(outputPath, []byte(file.GetContent()))
	if err != nil {
		slog.ErrorContext(ctx, "failed to write file", slog.Any("error", err))
		return
	}
	slog.Info("file written", slog.String("path", outputPath))
}

func generateCode(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	validatedRequest, err := protoplugin.NewRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate request: %w", err)
	}

	plugin, err := protogen.Options{}.New(validatedRequest.CodeGeneratorRequest())
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}
	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}
		gengo.GenerateFile(plugin, f) // protoc-gen-go
	}
	response := plugin.Response()
	if response.Error != nil {
		return nil, fmt.Errorf("failed to generate files: %w", errors.New(response.GetError()))
	}

	return response, nil
}

func writeToDisk(outputPath string, content []byte) error {
	err := os.MkdirAll(filepath.Dir(outputPath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	err = os.WriteFile(outputPath, content, 0o640) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
