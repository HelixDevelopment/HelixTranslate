// Command grpc-translate-probe is a standalone real-service gRPC client that
// drives a full async translate round-trip against a live TranslationService:
// StartTranslation(input_file, output_file) -> poll GetTranslationStatus to a
// terminal state -> print the resulting status + generated-file paths.
//
// It performs NO file I/O on the server's behalf: the input file must already
// exist on the gRPC server's filesystem at -input (stage it via `podman cp`),
// and the translated output is read back out-of-band (e.g. `podman exec cat`)
// from a path reported in the completed status's Files[]. This keeps the probe
// a pure wire-protocol client (§11.4.27 real-system, §11.4.107 real content).
//
// Usage:
//
//	grpc-translate-probe -addr nezha.local:50061 \
//	  -input /tmp/in.txt -output /tmp/out.epub \
//	  -src ru -dst sr -provider deepseek [-script latin] [-timeout 8m]
//
// Exit codes: 0 completed; 1 failed/cancelled/error; 2 usage/dial error.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "digital.vasic.translator/pkg/grpc/proto"
)

func main() {
	addr := flag.String("addr", "nezha.local:50061", "gRPC server address")
	input := flag.String("input", "", "input file path ON THE SERVER (required)")
	output := flag.String("output", "", "output file path ON THE SERVER (required)")
	src := flag.String("src", "ru", "source language")
	dst := flag.String("dst", "sr", "target language")
	provider := flag.String("provider", "deepseek", "provider type (provider_config.type)")
	model := flag.String("model", "", "model (optional)")
	script := flag.String("script", "", "target script: latin|cyrillic|\"\"")
	session := flag.String("session", "", "session id (default: time-based)")
	timeout := flag.Duration("timeout", 8*time.Minute, "overall poll timeout")
	flag.Parse()

	if *input == "" || *output == "" {
		log.Printf("ERROR: -input and -output are required (server-side paths)")
		flag.Usage()
		os.Exit(2)
	}
	if *session == "" {
		*session = fmt.Sprintf("probe-%d", time.Now().UnixNano())
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("ERROR: dial %s: %v", *addr, err)
		os.Exit(2)
	}
	defer conn.Close()
	client := pb.NewTranslationServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout+30*time.Second)
	defer cancel()

	req := &pb.TranslationRequest{
		SessionId:  *session,
		InputFile:  *input,
		OutputFile: *output,
		SourceLang: *src,
		TargetLang: *dst,
		Script:     *script,
		ProviderConfig: &pb.ProviderConfig{
			Type:  *provider,
			Model: *model,
		},
		Options: &pb.TranslationOptions{},
	}

	start := time.Now()
	resp, err := client.StartTranslation(ctx, req)
	if err != nil {
		log.Printf("ERROR: StartTranslation: %v", err)
		os.Exit(1)
	}
	fmt.Printf("START session=%s status=%s msg=%q\n", resp.GetSessionId(), resp.GetStatus(), resp.GetMessage())
	if resp.GetStatus() == "error" {
		os.Exit(1)
	}

	deadline := time.Now().Add(*timeout)
	for {
		if time.Now().After(deadline) {
			log.Printf("ERROR: poll timeout after %s (last poll not terminal)", *timeout)
			os.Exit(1)
		}
		time.Sleep(3 * time.Second)
		st, serr := client.GetTranslationStatus(ctx, &pb.TranslationStatusRequest{SessionId: *session})
		if serr != nil {
			log.Printf("WARN: GetTranslationStatus: %v", serr)
			continue
		}
		fmt.Printf("POLL [%6.1fs] status=%-9s progress=%5.1f%% step=%q\n",
			time.Since(start).Seconds(), st.GetStatus(), st.GetProgressPercentage(), st.GetCurrentStep())

		switch st.GetStatus() {
		case "completed":
			fmt.Printf("COMPLETED in %s\n", time.Since(start).Round(time.Second))
			for _, f := range st.GetFiles() {
				fmt.Printf("FILE type=%s path=%s\n", f.GetType(), f.GetPath())
			}
			os.Exit(0)
		case "failed", "cancelled":
			fmt.Printf("TERMINAL status=%s error=%q\n", st.GetStatus(), st.GetErrorMessage())
			os.Exit(1)
		}
	}
}
