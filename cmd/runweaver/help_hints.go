package main

func commandHint(command string) string {
	switch command {
	case "scan":
		return "runweaver scan --repo <path> [--out file]"
	case "index":
		return "runweaver index --repo <path> [--changed-only] [--prune] [--classification deterministic]"
	case "index clean":
		return "runweaver index clean --repo <path>"
	case "classify":
		return "runweaver classify --repo <path> [--classification auto|ai|deterministic] [--apply]"
	case "refresh":
		return "runweaver refresh --repo <path> [--apply] [--classification auto|ai|deterministic]"
	case "doctor":
		return "runweaver doctor --repo <path>"
	case "doctor model":
		return "runweaver doctor model --repo <path> --provider openai-compatible"
	case "doctor opencode":
		return "runweaver doctor opencode --repo <path> [--skip-model-check] [--timeout 60s]"
	case "doctor runtime":
		return "runweaver doctor runtime --repo <path> [--runtime all]"
	case "doctor processes":
		return "runweaver doctor processes [--summary]"
	case "init":
		return "runweaver init --repo <path> [--runtime opencode|codex|claude|all] [--force] [--require-model] [--classification auto|ai|deterministic]"
	case "workflow run":
		return "runweaver workflow run --workflow <file> --task <text> [--repo <path>] [--runtime opencode|codex|claude] [--execute]"
	case "workflow update":
		return "runweaver workflow update --repo <path> --resume latest --phase <id> --status in_progress"
	case "workflow verify":
		return "runweaver workflow verify --repo <path> --resume latest"
	default:
		return "run runweaver help"
	}
}
