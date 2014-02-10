package build

type runner struct {
	execFile string
}

func (r *runner) Run(args []string) (err error) {
	cmd := []string{r.execFile}
	cmd = append(cmd, args...)
	err = execCmd(cmd...)
	return
}
