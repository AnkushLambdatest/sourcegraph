package runner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/database/migration/definition"
)

type SchemaOutOfDateError struct {
	schemaName      string
	missingVersions []int
}

func (e *SchemaOutOfDateError) Error() string {
	return (instructionalError{
		class: "schema out of date",
		description: fmt.Sprintf(
			"schema %q requires the following migrations to be applied: %s\n",
			e.schemaName,
			strings.Join(intsToStrings(e.missingVersions), ", "),
		),
		instructions: strings.Join([]string{
			`This software expects a migrator instance to have run on this schema prior to the deployment of this process.`,
			`If this error is occurring directly after an upgrade, roll back your instance to the previous versiona nd ensure the migrator instance runs successfully prior attempting to re-upgrade.`,
		}, " "),
	}).Error()
}

func newOutOfDateError(schemaContext schemaContext, schemaVersion schemaVersion) error {
	definitions, err := schemaContext.schema.Definitions.Up(
		schemaVersion.appliedVersions,
		extractIDs(schemaContext.schema.Definitions.Leaves()),
	)
	if err != nil {
		return err
	}

	return &SchemaOutOfDateError{
		schemaName:      schemaContext.schema.Name,
		missingVersions: extractIDs(definitions),
	}
}

type dirtySchemaError struct {
	schemaName    string
	dirtyVersions []definition.Definition
}

func newDirtySchemaError(schemaName string, definitions []definition.Definition) error {
	return &dirtySchemaError{
		schemaName:    schemaName,
		dirtyVersions: definitions,
	}
}

func (e *dirtySchemaError) Error() string {
	return (instructionalError{
		class: "dirty database",
		description: fmt.Sprintf(
			"schema %q marked the following migrations as failed: %s\n",
			e.schemaName,
			strings.Join(intsToStrings(extractIDs(e.dirtyVersions)), ", "),
		),

		instructions: strings.Join([]string{
			`The target schema is marked as dirty and no other migration operation is seen running on this schema.`,
			`The last migration operation over this schema has failed (or, at least, the migrator instance issuing that migration has died).`,
			`Please contact support@sourcegraph.com for further assistance.`,
		}, " "),
	}).Error()
}

type privilegedMigrationError struct {
	schemaName  string
	definitions []definition.Definition
}

func newPrivilegedMigrationError(schemaName string, definitions []definition.Definition) error {
	return &privilegedMigrationError{
		schemaName:  schemaName,
		definitions: definitions,
	}
}

func (e *privilegedMigrationError) Error() string {
	var strIDs []string
	for _, definition := range e.definitions {
		strIDs = append(strIDs, strconv.Itoa(definition.ID))
	}

	migrationListText := ""
	if len(strIDs) == 1 {
		migrationListText = fmt.Sprintf("migration %s", strIDs[0])
	} else {
		strIDs[len(strIDs)-1] = "and " + strIDs[len(strIDs)-1]
		migrationListText = fmt.Sprintf("migrations %s", strings.Join(strIDs, ", "))
	}

	return (instructionalError{
		class: "refusing to apply a privileged migration",
		description: fmt.Sprintf(
			"schema %q requires database %s to be applied by a database user with elevated permissions\n",
			e.schemaName,
			migrationListText,
		),
		instructions: strings.Join([]string{
			`The migration runner is currently being run with -unprivileged-only.`,
			`The indicated migration is marked as privileged and cannot be applied by this invocation of the migration runner.`,
			`Before re-invoking the migration runner, follow the instructions on https://docs.sourcegraph.com/admin/how-to/privileged_migrations.`,
			`Please contact support@sourcegraph.com for further assistance.`,
		}, " "),
	}).Error()
}

type instructionalError struct {
	class        string
	description  string
	instructions string
}

func (e instructionalError) Error() string {
	return fmt.Sprintf("%s: %s\n\n%s\n", e.class, e.description, e.instructions)
}
