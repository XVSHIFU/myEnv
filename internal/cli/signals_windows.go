package cli

import "os"

func operationSignals() []os.Signal { return []os.Signal{os.Interrupt} }
