package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tpr/backend/data"
)

type Subscription struct {
	URL string `json:"url"`
}

var CreateSubscription = Define(ActionDef[Subscription, Empty]{Name: "create_subscription", Description: "Subscribe to a feed", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p Subscription) (Empty, error) {
	if p.URL == "" {
		return Empty{}, invalid(`Request must include the attribute "url"`)
	}
	if err := data.InsertSubscription(ctx, c.pool, User(ctx).ID.Int32, p.URL); err != nil {
		return Empty{}, invalid("Bad user name or password")
	}
	return Empty{}, nil
}})

type ID struct {
	ID int32 `json:"id"`
}

var DeleteSubscription = Define(ActionDef[ID, Empty]{Name: "delete_subscription", Description: "Unsubscribe from a feed", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p ID) (Empty, error) {
	err := data.DeleteSubscription(ctx, c.pool, User(ctx).ID.Int32, p.ID)
	if err != nil {
		return Empty{}, invalid(fmt.Sprintf("Error deleting subscription: %v", err))
	}
	return Empty{}, nil
}})
var GetFeeds = Define(ActionDef[Empty, json.RawMessage]{Name: "get_feeds", Description: "List the reader's feeds", Handler: func(ctx context.Context, c *Core, p Empty) (json.RawMessage, error) {
	var b bytes.Buffer
	err := data.CopySubscriptionsForUserAsJSON(ctx, c.pool, &b, User(ctx).ID.Int32)
	return json.RawMessage(b.Bytes()), err
}})
var GetUnreadItems = Define(ActionDef[Empty, json.RawMessage]{Name: "get_unread_items", Description: "List unread articles", Handler: func(ctx context.Context, c *Core, p Empty) (json.RawMessage, error) {
	var b bytes.Buffer
	err := data.CopyUnreadItemsAsJSONByUserID(ctx, c.pool, &b, User(ctx).ID.Int32)
	return json.RawMessage(b.Bytes()), err
}})
var GetArchivedItems = Define(ActionDef[Empty, json.RawMessage]{Name: "get_archived_items", Description: "List recent archived articles", Handler: func(ctx context.Context, c *Core, p Empty) (json.RawMessage, error) {
	var b bytes.Buffer
	err := data.CopyArchivedItemsAsJSONByUserID(ctx, c.pool, &b, User(ctx).ID.Int32)
	return json.RawMessage(b.Bytes()), err
}})
var MarkItemRead = Define(ActionDef[ID, Empty]{Name: "mark_item_read", Description: "Mark an article read", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p ID) (Empty, error) {
	err := data.MarkItemRead(ctx, c.pool, User(ctx).ID.Int32, p.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return Empty{}, err
}})

type ItemIDs struct {
	ItemIDs []int32 `json:"itemIDs"`
}

var MarkMultipleItemsRead = Define(ActionDef[ItemIDs, Empty]{Name: "mark_multiple_items_read", Description: "Mark multiple articles read", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p ItemIDs) (Empty, error) {
	for _, id := range p.ItemIDs {
		if err := data.MarkItemRead(ctx, c.pool, User(ctx).ID.Int32, id); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Empty{}, err
		}
	}
	return Empty{}, nil
}})

type ImportResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Success bool   `json:"success"`
}

var ImportFeeds = Define(ActionDef[OpmlDocument, []ImportResult]{Name: "import_feeds", Description: "Subscribe to feeds from an OPML document", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p OpmlDocument) ([]ImportResult, error) {
	results := make([]ImportResult, 0, len(p.Body.Outlines))
	for _, o := range p.Body.Outlines {
		err := data.InsertSubscription(ctx, c.pool, User(ctx).ID.Int32, o.URL)
		results = append(results, ImportResult{o.Title, o.URL, err == nil})
	}
	return results, nil
}})
var ExportFeeds = Define(ActionDef[Empty, OpmlDocument]{Name: "export_feeds", Description: "Export subscriptions as an OPML document", Handler: func(ctx context.Context, c *Core, p Empty) (OpmlDocument, error) {
	u := User(ctx)
	subs, err := data.SelectSubscriptions(ctx, c.pool, u.ID.Int32)
	if err != nil {
		return OpmlDocument{}, err
	}
	doc := OpmlDocument{Version: "1.0"}
	doc.Head.Title = "The Pithy Reader Export for " + u.Name.String
	for _, s := range subs {
		doc.Body.Outlines = append(doc.Body.Outlines, OpmlOutline{Text: s.Name.String, Title: s.Name.String, Type: "rss", URL: s.URL.String})
	}
	return doc, nil
}})
