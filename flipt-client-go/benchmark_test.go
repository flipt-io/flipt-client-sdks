//go:build benchmarks
// +build benchmarks

package flipt_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"

	flipt "go.flipt.io/flipt-client"
)

var (
	benchClient *flipt.Client
	cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile  = flag.String("memprofile", "", "write memory profile to file")
)

func TestMain(m *testing.M) {
	opts := []flipt.Option{}

	if os.Getenv("FLIPT_URL") != "" {
		opts = append(opts, flipt.WithURL(os.Getenv("FLIPT_URL")))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(os.Getenv("FLIPT_AUTH_TOKEN")))
	}

	var err error
	benchClient, err = flipt.NewClient(
		context.TODO(),
		opts...,
	)
	if err != nil {
		panic(err)
	}

	flag.Parse()

	// CPU profiling
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	// Run tests/benchmarks
	code := m.Run()

	// Memory profiling
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = f.Close()
		}()
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
	}

	os.Exit(code)
}

func generateLargeContext(size int) map[string]string {
	context := make(map[string]string)
	for i := range size {
		context[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	return context
}

func BenchmarkVariantEvaluation(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateVariant(ctx, &flipt.EvaluationRequest{FlagKey: "flag1", EntityID: "someentity", Context: map[string]string{
				"fizz": "buzz",
			}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MediumContext", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(10)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateVariant(ctx, &flipt.EvaluationRequest{FlagKey: "flag1", EntityID: "someentity", Context: context})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeContext", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(100)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateVariant(ctx, &flipt.EvaluationRequest{FlagKey: "flag1", EntityID: "someentity", Context: context})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBooleanEvaluation(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBoolean(ctx, &flipt.EvaluationRequest{FlagKey: "flag_boolean", EntityID: "someentity", Context: map[string]string{
				"fizz": "buzz",
			}})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MediumContext", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(10)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBoolean(ctx, &flipt.EvaluationRequest{FlagKey: "flag_boolean", EntityID: "someentity", Context: context})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeContext", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(100)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBoolean(ctx, &flipt.EvaluationRequest{FlagKey: "flag_boolean", EntityID: "someentity", Context: context})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBatchEvaluation(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		ctx := context.Background()
		requests := []*flipt.EvaluationRequest{
			{
				FlagKey:  "flag1",
				EntityID: "someentity",
				Context: map[string]string{
					"fizz": "buzz",
				},
			},
			{
				FlagKey:  "flag_boolean",
				EntityID: "someentity",
				Context: map[string]string{
					"fizz": "buzz",
				},
			},
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBatch(ctx, requests)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MediumBatch", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(10)
		requests := make([]*flipt.EvaluationRequest, 10)
		for i := range requests {
			requests[i] = &flipt.EvaluationRequest{
				FlagKey:  fmt.Sprintf("flag%d", i),
				EntityID: "someentity",
				Context:  context,
			}
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBatch(ctx, requests)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeBatch", func(b *testing.B) {
		ctx := context.Background()
		context := generateLargeContext(100)
		requests := make([]*flipt.EvaluationRequest, 50)
		for i := range requests {
			requests[i] = &flipt.EvaluationRequest{
				FlagKey:  fmt.Sprintf("flag%d", i),
				EntityID: "someentity",
				Context:  context,
			}
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := benchClient.EvaluateBatch(ctx, requests)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkListFlags(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchClient.ListFlags(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
