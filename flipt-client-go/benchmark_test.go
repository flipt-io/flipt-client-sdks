package evaluation_test

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
	benchClient *flipt.EvaluationClient
	cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile  = flag.String("memprofile", "", "write memory profile to file")
)

func TestMain(m *testing.M) {
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
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
	}

	os.Exit(code)
}

func init() {
	opts := []flipt.ClientOption{}

	if os.Getenv("FLIPT_URL") != "" {
		opts = append(opts, flipt.WithURL(os.Getenv("FLIPT_URL")))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(os.Getenv("FLIPT_AUTH_TOKEN")))
	}

	var err error
	benchClient, err = flipt.NewEvaluationClient(
		context.TODO(),
		opts...,
	)
	if err != nil {
		panic(err)
	}
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
			_, err := benchClient.EvaluateVariant(ctx, "flag1", "someentity", map[string]string{
				"fizz": "buzz",
			})
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
			_, err := benchClient.EvaluateVariant(ctx, "flag1", "someentity", context)
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
			_, err := benchClient.EvaluateVariant(ctx, "flag1", "someentity", context)
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
			_, err := benchClient.EvaluateBoolean(ctx, "flag_boolean", "someentity", map[string]string{
				"fizz": "buzz",
			})
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
			_, err := benchClient.EvaluateBoolean(ctx, "flag_boolean", "someentity", context)
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
			_, err := benchClient.EvaluateBoolean(ctx, "flag_boolean", "someentity", context)
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
				EntityId: "someentity",
				Context: map[string]string{
					"fizz": "buzz",
				},
			},
			{
				FlagKey:  "flag_boolean",
				EntityId: "someentity",
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
				EntityId: "someentity",
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
				EntityId: "someentity",
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
