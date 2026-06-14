package grpc

import (
	"sync"
	"testing"

	"digital.vasic.translator/pkg/grpc/proto"
)

// --- Bug F: CoreTranslatorImpl job-field data race --------------------------
//
// CoreTranslatorImpl.GetStatus and Cancel read/write the job's mutable fields
// (Status, Progress, Step, Steps, Files) under ct.mutex. But the translation
// pipeline mutates those SAME fields with NO lock held:
//
//	Translate -> executeTranslationPipeline -> updateJobStep / addGeneratedFile /
//	completeStep / failStep
//
// updateJobStep does `job.Steps = append(job.Steps, step)` + writes job.Progress
// + job.Step; addGeneratedFile does `job.Files = append(job.Files, file)`. None of
// these takes ct.mutex. Meanwhile the gRPC read path (Server.GetTranslationStatus
// -> translator.GetStatus) reads job.Steps / job.Files / job.Progress / job.Step
// UNDER ct.mutex.RLock(). A status poll arriving while the pipeline is running is
// a textbook concurrent read+write on the same slice header / float64 / string —
// the running translation and a status RPC are inherently concurrent (the whole
// point of GetStatus is to poll an in-flight job).
//
// RED (pre-fix): `go test -race` reports a DATA RACE between updateJobStep/
// addGeneratedFile (unlocked write) and GetStatus (locked read).
// GREEN (post-fix): the pipeline mutations take ct.mutex, so reader and writer
// are serialised and -race is clean.
func TestCoreTranslator_GetStatus_RaceWithPipelineMutations(t *testing.T) {
	ct := newCT()

	// Register a running job, exactly as Translate does before running the pipeline.
	job := &TranslationJob{
		ID:     "race-job",
		Status: "running",
		Steps:  make([]*proto.TranslationStep, 0),
		Files:  make([]*proto.GeneratedFile, 0),
	}
	ct.mutex.Lock()
	ct.sessions["race-job"] = job
	ct.mutex.Unlock()

	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine: the UNLOCKED pipeline mutations (the production code path
	// inside executeTranslationPipeline runs these with no ct.mutex held).
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			ct.updateJobStep(job, ct.createStep("step", "running"))
			ct.addGeneratedFile(job, "/out/f.md", "translated_md", 10, true, "ok")
		}
	}()

	// Reader goroutine: the LOCKED status RPC path. Concurrent with the running
	// pipeline this races on job.Steps / job.Files / job.Progress / job.Step.
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			resp, err := ct.GetStatus("race-job")
			if err != nil {
				t.Errorf("GetStatus: %v", err)
				return
			}
			// Touch the read fields so the race detector observes the read.
			_ = resp.GetProgressPercentage()
			_ = len(resp.GetSteps())
			_ = len(resp.GetFiles())
			_ = resp.GetCurrentStep()
		}
	}()

	wg.Wait()
}
