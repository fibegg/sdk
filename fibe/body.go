package fibe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

const (
	maxResponseBody = int64(10 * 1024 * 1024)
	maxErrorBody    = int64(1 * 1024 * 1024)
)

func readLimited(r io.Reader, limit int64, label string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("fibe: %s exceeds %d bytes", label, limit)
	}
	return data, nil
}

func decodeJSONLimitedProjected(ctx context.Context, r io.Reader, limit int64, label string, dst any) error {
	value := reflect.ValueOf(dst)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		if err := decodeJSONLimited(r, limit, label, dst); err != nil {
			return err
		}
		return applyProjection(ctx, dst)
	}
	temporary := reflect.New(value.Elem().Type())
	if err := decodeJSONLimited(r, limit, label, temporary.Interface()); err != nil {
		return err
	}
	if err := applyProjection(ctx, temporary.Interface()); err != nil {
		return err
	}
	value.Elem().Set(temporary.Elem())
	return nil
}

func decodeJSONLimited(r io.Reader, limit int64, label string, dst any) error {
	data, err := readLimited(r, limit, label)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("fibe: %s contains multiple JSON values", label)
		}
		return fmt.Errorf("fibe: %s contains trailing data: %w", label, err)
	}
	return nil
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.CopyN(io.Discard, body, 64*1024)
	_ = body.Close()
}
