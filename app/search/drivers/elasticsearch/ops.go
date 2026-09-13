package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
)

func indexJSON(ctx context.Context, es *elasticsearch.Client, index, documentID string, body []byte) error {
	res, err := es.Index(
		index,
		bytes.NewReader(body),
		es.Index.WithContext(ctx),
		es.Index.WithDocumentID(documentID),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("es index: %s: %s", res.Status(), string(b))
	}
	return nil
}

func indexValue(ctx context.Context, es *elasticsearch.Client, index, documentID string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return indexJSON(ctx, es, index, documentID, body)
}

func deleteDocument(ctx context.Context, es *elasticsearch.Client, index, documentID string) error {
	res, err := es.Delete(
		index,
		documentID,
		es.Delete.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("es delete: %s: %s", res.Status(), string(b))
	}
	return nil
}
