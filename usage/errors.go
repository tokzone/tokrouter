package usage

import "errors"

// ErrDisabled is returned when usage tracking is disabled
var ErrDisabled = errors.New(`usage tracking is disabled

Enable it in config.yaml:
  stats:
    enabled: true
    db_path: "./data/usage.db"`)
