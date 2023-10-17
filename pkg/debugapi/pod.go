package debugapi

type podProcessor struct {
}

func (p *podProcessor) Process() (string, error) {
	testdata := `{"we":"xxx", "xx":"xxx"}`
	return testdata, nil
}
