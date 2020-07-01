// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Program xtoproto infers .proto definitions from record-oriented files (CSV,
// XML, etc.).
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/xtoproto/service"
	"google.golang.org/protobuf/encoding/prototext"

	spb "github.com/google/xtoproto/proto/service"
)

const (
	outDirMode  os.FileMode = 0770
	outFileMode os.FileMode = 0660
)

var (
	cfg = registerFlags(flag.CommandLine)

	readFile service.FileReaderFunc = func(_ context.Context, path string) ([]byte, error) {
		return ioutil.ReadFile(path)
	}
	writeFile service.FileWriterFunc = func(_ context.Context, path string, data []byte) error {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, outDirMode); err != nil {
			return err
		}
		return ioutil.WriteFile(path, data, outFileMode)
	}
)

type config struct {
	defaultWorkspaceDir string
	csvPath             string
}

func registerFlags(fs *flag.FlagSet) *config {
	cfg := &config{}
	fs.StringVar(&cfg.defaultWorkspaceDir, "default_workspace", "/tmp/example-workspace", "default workspace directory")
	fs.StringVar(&cfg.csvPath, "csv", "", "path to input csv file")
	return cfg
}

func main() {
	flag.Parse()
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "fatal error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	s := service.New(cfg.defaultWorkspaceDir, readFile, writeFile)
	resp1, err := s.Infer(ctx, &spb.InferRequest{
		GoPackageName: "example",
		GoProtoImport: "not/sure",
		InputFormat:   spb.Format_CSV,
		MessageName:   "MyMessage",
		PackageName:   "mypackage",
		ExampleInputs: []*spb.InputFile{
			{
				Spec: &spb.InputFile_InputPath{
					InputPath: cfg.csvPath,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("InferResponse:\n%s\n", prototext.Format(resp1))
	resp2, err := s.GenerateCode(ctx, &spb.GenerateCodeRequest{
		Mapping: resp1.GetBestMappingCandidate().GetTopLevelMapping(),
		ProtoDefinition: &spb.GenerateCodeRequest_ProtoDefinition{
			Directory:        "generated",
			ProtoFileName:    "example.proto",
			UpdateBuildRules: true,
		},
		Converter: &spb.GenerateCodeRequest_Converter{
			Directory:        "generated",
			GoFileName:       "exampleconv.go",
			UpdateBuildRules: true,
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("GenerateCodeResponse:\n%s\n", prototext.Format(resp2))

	return nil
}
