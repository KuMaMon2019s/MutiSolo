package controllayer

type PipelineResult struct {
	Artifact Artifact `json:"artifact"`
	Path     string   `json:"path"`
}

func RunPipeline(input PipelineInput, artifactDir string) (PipelineResult, error) {
	generation, err := Generate(input.Prompt)
	if err != nil {
		return PipelineResult{}, err
	}
	validation := Validate(input.Prompt, generation, input.ApproveSystem)
	artifact := Artifact{
		ID:         NewArtifactID(input.Prompt, generation, validation),
		Prompt:     input.Prompt,
		Generation: generation,
		Validation: validation,
	}
	path, err := StoreArtifact(artifactDir, artifact)
	if err != nil {
		return PipelineResult{}, err
	}
	return PipelineResult{
		Artifact: artifact,
		Path:     path,
	}, nil
}
