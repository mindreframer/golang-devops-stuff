package upstart

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var jobName string = "atd"

var runUpstartTests = false

func init() {
	if s := os.Getenv("TEST_JOB"); s != "" {
		jobName = s
	}

	c := exec.Command("which", "initctl")
	c.Run()
	runUpstartTests = c.ProcessState.Success()
}

func TestJobs(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	jobs, err := u.Jobs()
	if err != nil {
		panic(err)
	}

	if len(jobs) == 0 {
		t.Fatal("Unable to get jobs")
	}

	var job *Job

	for _, j := range jobs {
		name, err := j.Name()
		if err != nil {
			panic(err)
		}

		if name == jobName {
			job = j
		}
	}

	if job == nil {
		t.Fatalf("Unable to find job: %s", jobName)
	}
}

func TestJob(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	if job == nil {
		t.Fatalf("Unable to find job: %s", jobName)
	}
}

func TestJobName(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	name, err := job.Name()
	if err != nil {
		panic(err)
	}

	if name != jobName {
		t.Fatalf("job name didn't work properly: %s != %s", name, jobName)
	}
}

func TestJobPid(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	exp, err := job.Pid()
	if err != nil {
		panic(err)
	}

	bytes, err := exec.Command("pgrep", jobName).CombinedOutput()
	if err != nil {
		panic(err)
	}

	act, err := strconv.Atoi(strings.TrimSpace(string(bytes)))
	if err != nil {
		panic(err)
	}

	if exp != int32(act) {
		t.Fatalf("pid for job isn't correct: %d != %d", exp, act)
	}
}

func TestJobPidReturnsErrorWhenMultipleInstances(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job("network-interface")
	if err != nil {
		panic(err)
	}

	_, err = job.Pid()
	if err == nil {
		t.Fatal("Pid didn't return an error")
	}
}

func TestInstanceWithJob(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	inst, err := u.Instance(jobName)
	if err != nil {
		panic(err)
	}

	if inst == nil {
		t.Fatalf("Unable to find inst: %s", jobName)
	}
}

func TestInstanceWithJobAndInstance(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	instName := "network-interface/lo"

	inst, err := u.Instance(instName)
	if err != nil {
		panic(err)
	}

	if inst == nil {
		t.Fatalf("Unable to find inst: %s", instName)
	}
}

func TestInstancePid(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	inst, err := u.Instance(jobName)
	if err != nil {
		panic(err)
	}

	exp, err := inst.Pid()
	if err != nil {
		panic(err)
	}

	bytes, err := exec.Command("pgrep", jobName).CombinedOutput()
	if err != nil {
		panic(err)
	}

	act, err := strconv.Atoi(strings.TrimSpace(string(bytes)))
	if err != nil {
		panic(err)
	}

	if exp != int32(act) {
		t.Fatalf("pid for job isn't correct: %d != %d", exp, act)
	}
}

func TestInstanceRestart(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	inst, err := u.Instance(jobName)
	if err != nil {
		panic(err)
	}

	start, err := inst.Pid()
	if err != nil {
		panic(err)
	}

	err = inst.Restart()
	if err != nil {
		panic(err)
	}

	cur, err := inst.Pid()

	if start == cur {
		t.Fatalf("job did not restart. old:%d, new:%d", start, cur)
	}
}

func TestJobStart(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	start, err := job.Pid()
	if err != nil {
		panic(err)
	}

	err = job.Stop()
	if err != nil {
		panic(err)
	}

	inst, err := job.Start()
	if err != nil {
		panic(err)
	}

	cur, err := inst.Pid()

	if start == cur {
		t.Fatalf("job did not restart. old:%d, new:%d", start, cur)
	}

	bytes, err := exec.Command("pgrep", jobName).CombinedOutput()
	if err != nil {
		panic(err)
	}

	act, err := strconv.Atoi(strings.TrimSpace(string(bytes)))
	if err != nil {
		panic(err)
	}

	if cur != int32(act) {
		t.Fatalf("pid for job isn't correct: %d != %d", cur, act)
	}
}

func TestJobRestart(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	job, err := u.Job(jobName)
	if err != nil {
		panic(err)
	}

	start, err := job.Pid()
	if err != nil {
		panic(err)
	}

	inst, err := job.Restart()
	if err != nil {
		panic(err)
	}

	cur, err := inst.Pid()

	if start == cur {
		t.Fatalf("job did not restart. old:%d, new:%d", start, cur)
	}

	bytes, err := exec.Command("pgrep", jobName).CombinedOutput()
	if err != nil {
		panic(err)
	}

	act, err := strconv.Atoi(strings.TrimSpace(string(bytes)))
	if err != nil {
		panic(err)
	}

	if cur != int32(act) {
		t.Fatalf("pid for job isn't correct: %d != %d", cur, act)
	}
}

func TestEmitEvent(t *testing.T) {
	if !runUpstartTests {
		t.SkipNow()
	}

	u, err := Dial()
	if err != nil {
		panic(err)
	}

	u.EmitEvent("test-booted", []string{}, true)
}
