package expr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/grafana/grafana/pkg/apimachinery/errutil"
	"github.com/grafana/grafana/pkg/expr/mathexp"
	"github.com/grafana/grafana/pkg/expr/sql"
	"github.com/grafana/grafana/pkg/infra/tracing"
)

// SQLCommand is an expression to run SQL over results
type SQLCommand struct {
	query       string
	varsToQuery []string
	refID       string
	limit       int64
}

// NewSQLCommand creates a new SQLCommand.
func NewSQLCommand(refID, rawSQL string) (*SQLCommand, error) {
	if rawSQL == "" {
		return nil, errutil.BadRequest("sql-missing-query",
			errutil.WithPublicMessage("missing SQL query"))
	}
	tables, err := sql.TablesList(rawSQL)
	if err != nil {
		logger.Warn("invalid sql query", "sql", rawSQL, "error", err)
		return nil, errutil.BadRequest("sql-invalid-sql",
			errutil.WithPublicMessage(fmt.Sprintf("invalid SQL query: %s", err)),
		)
	}
	if len(tables) == 0 {
		logger.Warn("no tables found in SQL query", "sql", rawSQL)
	}
	if tables != nil {
		logger.Debug("REF tables", "tables", tables, "sql", rawSQL)
	}

	defaultLimit := int64(20)
	return &SQLCommand{
		query:       rawSQL,
		varsToQuery: tables,
		refID:       refID,
		limit:       defaultLimit,
	}, nil
}

// UnmarshalSQLCommand creates a SQLCommand from Grafana's frontend query.
func UnmarshalSQLCommand(rn *rawNode) (*SQLCommand, error) {
	if rn.TimeRange == nil {
		logger.Error("time range must be specified for refID", "refID", rn.RefID)
		return nil, fmt.Errorf("time range must be specified for refID %s", rn.RefID)
	}

	expressionRaw, ok := rn.Query["expression"]
	if !ok {
		logger.Error("no expression in the query", "query", rn.Query)
		return nil, errors.New("no expression in the query")
	}
	expression, ok := expressionRaw.(string)
	if !ok {
		logger.Error("expected sql expression to be type string", "expression", expressionRaw)
		return nil, fmt.Errorf("expected sql expression to be type string, but got type %T", expressionRaw)
	}

	return NewSQLCommand(rn.RefID, expression)
}

// NeedsVars returns the variable names (refIds) that are dependencies
// to execute the command and allows the command to fulfill the Command interface.
func (cmd *SQLCommand) NeedsVars() []string {
	return cmd.varsToQuery
}

// Execute runs the command and returns the results or an error if the command
// failed to execute.
func (cmd *SQLCommand) Execute(ctx context.Context, now time.Time, vars mathexp.Vars, tracer tracing.Tracer) (mathexp.Results, error) {
	_, span := tracer.Start(ctx, "SSE.ExecuteSQL")
	defer span.End()

	allFrames := []*data.Frame{}
	for _, ref := range cmd.varsToQuery {
		results, ok := vars[ref]
		if !ok {
			logger.Warn("no results found for", "ref", ref)
			continue
		}
		frames := results.Values.AsDataFrames(ref)
		allFrames = append(allFrames, frames...)
	}

	totalRows := totalRows(allFrames)
	// limit of 0 or less means no limit (following convention)
	if cmd.limit > 0 && totalRows > cmd.limit {
		return mathexp.Results{},
			fmt.Errorf(
				"SQL expression: total row count across all input tables exceeds limit of %d. Total rows: %d",
				cmd.limit,
				totalRows,
			)
	}

	logger.Debug("Executing query", "query", cmd.query, "frames", len(allFrames))

	db := sql.DB{}
	frame, err := db.QueryFrames(ctx, cmd.refID, cmd.query, allFrames)

	rsp := mathexp.Results{}
	if err != nil {
		logger.Error("Failed to query frames", "error", err.Error())
		rsp.Error = err
		return rsp, nil
	}
	logger.Debug("Done Executing query", "query", cmd.query, "rows", frame.Rows())

	if frame.Rows() == 0 {
		rsp.Values = mathexp.Values{
			mathexp.NoData{Frame: frame},
		}
		return rsp, nil
	}

	rsp.Values = mathexp.Values{
		mathexp.TableData{Frame: frame},
	}

	return rsp, nil
}

func (cmd *SQLCommand) Type() string {
	return TypeSQL.String()
}

func totalRows(frames []*data.Frame) int64 {
	total := 0
	for _, frame := range frames {
		if frame != nil {
			total += frame.Rows()
		}
	}
	return int64(total)
}
