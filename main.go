package main

import (
	"context"
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

	if len(os.Args) == 1 || os.Args[1] != "generate" {
		os.Args = append(os.Args, "--help")
	} else {
		os.Args = os.Args[1:] // remove "generate" from the args
	}

	cfg := &Config{}
	structconf.MustLoadAndValidate(cfg,
		"tilebox generate",
		structconf.WithDescription("Generate Tilebox datasets types for Go"),
	)

	client := tileboxdatasets.NewClient(
		tileboxdatasets.WithAPIKey(cfg.TileboxAPIKey),
	)
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

	// protoplugin validates the request
	request, err := protoplugin.NewRequest(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{datasetFileDescriptor.GetName()},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto),
			protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto),
			protodesc.ToFileDescriptorProto(datasetsv1.File_datasets_v1_well_known_types_proto),
			datasetFileDescriptor,
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create request", slog.Any("error", err))
		return
	}

	plugin, err := protogen.Options{}.New(request.CodeGeneratorRequest())
	if err != nil {
		slog.ErrorContext(ctx, "failed to create plugin", slog.Any("error", err))
		return
	}
	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}
		gengo.GenerateFile(plugin, f) // protoc-gen-go
	}
	response := plugin.Response()
	if response.Error != nil {
		slog.ErrorContext(ctx, "failed to generate files", slog.Any("error", response.GetError()))
		return
	}

	file := response.GetFile()[0] // we only generate one file
	err = os.MkdirAll(filepath.Join(cfg.Out, filepath.Dir(file.GetName())), 0o755)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create output directory", slog.Any("error", err))
		return
	}

	outputFile := filepath.Join(cfg.Out, file.GetName())
	err = os.WriteFile(outputFile, []byte(file.GetContent()), 0o640) //nolint:gosec
	if err != nil {
		slog.ErrorContext(ctx, "failed to write file", slog.Any("error", err))
		return
	}

	slog.InfoContext(ctx, "file written", slog.String("path", outputFile))
}
