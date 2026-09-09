package core

type SyncPhase string

const (
	PhaseWaiting      SyncPhase = "waiting"
	PhaseResolving    SyncPhase = "resolving"
	PhaseBackend      SyncPhase = "backend"
	PhaseSDK          SyncPhase = "sdk"
	PhaseNode         SyncPhase = "node"
	PhasePython       SyncPhase = "python"
	PhaseDependencies SyncPhase = "dependencies"
	PhaseVerifying    SyncPhase = "verifying"
	PhasePublishing   SyncPhase = "publishing"
)

func (r SyncRequest) phase(phase SyncPhase) {
	if r.Progress != nil {
		r.Progress(phase)
	}
}
