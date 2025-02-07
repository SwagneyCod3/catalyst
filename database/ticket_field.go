package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/arangodb/go-driver"
	"github.com/iancoleman/strcase"
	"github.com/mingrammer/commonregex"

	"github.com/SecurityBrewery/catalyst/bus"
	"github.com/SecurityBrewery/catalyst/database/busdb"
	"github.com/SecurityBrewery/catalyst/generated/model"
	"github.com/SecurityBrewery/catalyst/generated/pointer"
	"github.com/SecurityBrewery/catalyst/generated/time"
)

func (db *Database) AddArtifact(ctx context.Context, id int64, artifact *model.Artifact) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	if artifact.Status == nil {
		artifact.Status = pointer.String("unknown")
	}

	if artifact.Type == nil {
		artifact.Type = pointer.String(inferType(artifact.Name))
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "modified": @now, "artifacts": PUSH(NOT_NULL(d.artifacts, []), @artifact) } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"artifact": artifact, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func inferType(name string) string {
	switch {
	case commonregex.IPRegex.MatchString(name):
		return "ip"
	case commonregex.LinkRegex.MatchString(name):
		return "url"
	case commonregex.EmailRegex.MatchString(name):
		return "email"
	case commonregex.MD5HexRegex.MatchString(name):
		return "md5"
	case commonregex.SHA1HexRegex.MatchString(name):
		return "sha1"
	case commonregex.SHA256HexRegex.MatchString(name):
		return "sha256"
	}

	return "unknown"
}

func (db *Database) RemoveArtifact(ctx context.Context, id int64, name string) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	FOR a IN NOT_NULL(d.artifacts, [])
	FILTER a.name == @name
	LET newartifacts = REMOVE_VALUE(d.artifacts, a)
	UPDATE d WITH { "modified": @now, "artifacts": newartifacts } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"name": name, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) SetTemplate(ctx context.Context, id int64, schema string) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "schema": @schema } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"schema": schema}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) AddComment(ctx context.Context, id int64, comment *model.CommentForm) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	if comment.Creator == nil || *comment.Creator == "" {
		user, exists := busdb.UserFromContext(ctx)
		if !exists {
			return nil, errors.New("no user in context")
		}

		comment.Creator = pointer.String(user.ID)
	}

	if comment.Created == nil {
		comment.Created = pointer.Time(time.Now().UTC())
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "modified": @now, "comments": PUSH(NOT_NULL(d.comments, []), @comment) } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"comment": comment, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) RemoveComment(ctx context.Context, id int64, commentID int64) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "modified": @now, "comments": REMOVE_NTH(d.comments, @commentID) } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"commentID": commentID, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) SetReferences(ctx context.Context, id int64, references []*model.Reference) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "modified": @now, "references": @references } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"references": references, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) AddFile(ctx context.Context, id int64, file *model.File) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `LET d = DOCUMENT(@@collection, @ID)
	` + ticketFilterQuery + `
	UPDATE d WITH { "modified": @now, "files": APPEND(NOT_NULL(d.files, []), [@file]) } IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{"file": file, "now": time.Now().UTC()}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.DocumentID(fmt.Sprintf("%s/%d", TicketCollectionName, id)),
		},
	})
}

func (db *Database) AddTicketPlaybook(ctx context.Context, id int64, playbookTemplate *model.PlaybookTemplateForm) (*model.TicketWithTickets, error) {
	pb, err := toPlaybook(playbookTemplate)
	if err != nil {
		return nil, err
	}

	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	playbookID := strcase.ToKebab(pb.Name)
	if playbookTemplate.ID != nil {
		playbookID = *playbookTemplate.ID
	}

	parentTicket, err := db.TicketGet(ctx, id)
	if err != nil {
		return nil, err
	}

	query := `FOR d IN @@collection 
	` + ticketFilterQuery + `
	FILTER d._key == @ID
	LET newplaybook =  ZIP( [@playbookID], [@playbook] )
	LET newplaybooks = MERGE(NOT_NULL(d.playbooks, {}), newplaybook)
	LET newticket = MERGE(d, { "modified": @now, "playbooks": newplaybooks })
	REPLACE d WITH newticket IN @@collection
	RETURN NEW`
	ticket, err := db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{
		"playbook":   pb,
		"playbookID": findName(parentTicket.Playbooks, playbookID),
		"now":        time.Now().UTC(),
	}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.NewDocumentID(TicketCollectionName, fmt.Sprintf("%d", id)),
		},
	})
	if err != nil {
		return nil, err
	}

	if err := runRootTask(extractTicketResponse(ticket), playbookID, db); err != nil {
		return nil, err
	}

	return ticket, nil
}

func findName(playbooks map[string]*model.PlaybookResponse, name string) string {
	if _, ok := playbooks[name]; !ok {
		return name
	}

	for i := 0; ; i++ {
		try := fmt.Sprintf("%s%d", name, i)
		if _, ok := playbooks[try]; !ok {
			return try
		}
	}
}

func runRootTask(ticket *model.TicketResponse, playbookID string, db *Database) error {
	playbook := ticket.Playbooks[playbookID]

	var root *model.TaskResponse
	for _, task := range playbook.Tasks {
		if task.Order == 0 {
			root = task
		}
	}

	runNextTasks(ticket.ID, playbookID, root.Next, root.Data, ticket, db)

	return nil
}

func (db *Database) RemoveTicketPlaybook(ctx context.Context, id int64, playbookID string) (*model.TicketWithTickets, error) {
	ticketFilterQuery, ticketFilterVars, err := db.Hooks.TicketWriteFilter(ctx)
	if err != nil {
		return nil, err
	}

	query := `FOR d IN @@collection 
	` + ticketFilterQuery + `
	FILTER d._key == @ID
	LET newplaybooks = UNSET(d.playbooks, @playbookID)
	REPLACE d WITH MERGE(d, { "modified": @now, "playbooks": newplaybooks }) IN @@collection
	RETURN NEW`

	return db.ticketGetQuery(ctx, id, query, mergeMaps(map[string]any{
		"playbookID": playbookID,
		"now":        time.Now().UTC(),
	}, ticketFilterVars), &busdb.Operation{
		Type: bus.DatabaseEntryUpdated,
		Ids: []driver.DocumentID{
			driver.NewDocumentID(TicketCollectionName, fmt.Sprintf("%d", id)),
		},
	})
}
