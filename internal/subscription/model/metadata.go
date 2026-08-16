package model

// EnsureMetadata returns a non-nil metadata map for the supplied value.
// It keeps existing keys and values, only allocating when metadata is nil.
func EnsureMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	return metadata
}

// CloneMetadata creates a shallow copy suitable for storing in a request or
// response so callers cannot accidentally mutate shared internal state.
func CloneMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

// SetMetadataKey sets key to value on a possibly nil metadata map and returns
// the normalized map. This keeps metadata initialization explicit at call sites.
func SetMetadataKey(metadata map[string]string, key, value string) map[string]string {
	normalized := EnsureMetadata(metadata)
	normalized[key] = value
	return normalized
}

// MetadataKeys returns a stable, sorted list of metadata keys. It is used by
// repository code when normalizing persisted records so output is repeatable.
func MetadataKeys(metadata map[string]string) []string {
	if metadata == nil {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	return keys
}
