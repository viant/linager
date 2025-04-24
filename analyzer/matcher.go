package analyzer

import (
	"os"
)

type MatcherFn func(info os.FileInfo) bool
