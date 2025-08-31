package main

import (
	"github.com/iancoleman/strcase"
	"go.corp.spacelink.com/sdks/go/pgtypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type (
	ExtraField struct {
		StructPropName    string //Property name to be added into the table struct
		StructPropType    string //The full path type of the property (e.g. models.MyType)
		FkStructPropName  string //Struct prpoerty name that is used in the foreign key
		RefStructPropName string //Struct property name of the referenced table struct
		HasMany           bool   // A one-one or one-to-many relationship
		Pointer           bool   // Should the added property be a pointer
	}
)

var (
	typeMaps = map[string]string{
		"jsonb": "datatypes.JSONMap",
		"uuid":  "datatypes.UUID",
	}
	domainTypeMaps = map[string]string{
		"spacelink_identifier": "sl_datatypes.SpacelinkIdentifier",
	}
	namingStrategy = schema.NamingStrategy{IdentifierMaxLength: 64}
	extraFields    = map[string][]ExtraField{
		//"ticket_extended": {
		//	{
		//		StructPropName:    "Attachments",
		//		StructPropType:    "models.Attachment",
		//		FkStructPropName:  "TicketID",
		//		RefStructPropName: "TicketID",
		//		HasMany:           true,
		//		Pointer:           true,
		//	},
		//},
	}
	outPath            = "cmd/app/db"
	modelPkgPath       = "cmd/app/db/models"
	importPackagePaths = []string{
		"go.corp.spacelink.com/sdks/go/sl_datatypes",
		"go.corp.spacelink.com/sdks/go/pgtypes",
	}
	jsonTagOverridesByTable = map[string]map[string]string{
		//"ticket_extended": {
		//	"subject_fts": "-",
		//},
	}
)

func main() {
	//sqliteToGorm()
	//postgresToGorm()
}

func sqliteToGorm() {
	var db *gorm.DB
	var err error
	db, err = gorm.Open(sqlite.Open("dev/schema.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err.Error())
	}
	sqldb, _ := db.DB()
	err = sqldb.Ping()
	if err != nil {
		log.Fatal("Unable to ping database: " + err.Error())
	}
	g := gen.NewGenerator(gen.Config{
		OutPath:           "internal/app/db",
		ModelPkgPath:      "internal/app/db/models",
		WithUnitTest:      false,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		Mode:              gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})
	g.WithJSONTagNameStrategy(func(col string) (tag string) { return strcase.ToLowerCamel(col) })
	dtMaps := pgtypes.DataTypeMap()
	dtMaps["jsonb"] = dt("datatypes.JSONMap")
	dtMaps["uuid"] = dt("datatypes.UUID")
	g.WithDataTypeMap(dtMaps)
	g.WithImportPkgPath("github.com/google/uuid", "go.corp.spacelink.com/sdks/go/pgtypes")
	g.UseDB(db)
	g.ApplyBasic(
		g.GenerateAllTable()...,
	)
	g.Execute()
}

func postgresToGorm() {
	var db *gorm.DB
	var err error
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "service_template"
	}
	dsn := "host=" + host + " dbname=" + dbname + " sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn))
	if err != nil {
		log.Fatal(err.Error())
	}
	sqldb, _ := db.DB()
	err = sqldb.Ping()
	if err != nil {
		log.Fatal("Unable to ping database: " + err.Error())
	}

	cleanUp()

	g := gen.NewGenerator(gen.Config{
		OutPath:           outPath,
		ModelPkgPath:      modelPkgPath,
		WithUnitTest:      false,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		Mode:              gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	tables := tableNames(db)
	materializedViews := materializedViewNames(db)

	g.WithJSONTagNameStrategy(func(col string) (tag string) { return strcase.ToLowerCamel(col) })
	g.WithImportPkgPath(importPackagePaths...)
	dtMaps := pgtypes.DataTypeMap()
	for k, v := range typeMaps {
		dtMaps[k] = dt(v)
	}
	dtMaps["text"] = func(columnType gorm.ColumnType) string {
		if colType, ok := columnType.ColumnType(); ok {
			if domain, ok := domainTypeMaps[colType]; ok {
				return domain
			}
		}
		return "string"
	}
	g.WithDataTypeMap(dtMaps)
	g.UseDB(db)
	modelsMap := map[string]any{}
	for _, tableName := range tables {
		model := g.GenerateModel(tableName)
		if ef, ok := extraFields[tableName]; ok {
			for _, ef := range ef {
				a := gen.FieldNew("", "", nil)
				f := a(nil)
				genRelationField(&ef, gen.Field(f))
				model.Fields = append(model.Fields, f)
			}
		}
		if jsonTagOverrides, ok := jsonTagOverridesByTable[tableName]; ok {
			for _, f := range model.Fields {
				if jsonTag, ok := jsonTagOverrides[f.ColumnName]; ok {
					f.Tag.Set("json", jsonTag)
				} else if jsonTag, ok := jsonTagOverrides[f.Name]; ok {
					f.Tag.Set("json", jsonTag)
				}
			}
		}
		modelsMap[tableName] = model
	}

	for _, viewName := range materializedViews {
		tmpViewName := viewName + "_temp"
		_, _ = sqldb.Query("drop view if exists " + tmpViewName)
		_, err = sqldb.Query("create view " + tmpViewName + " as select * from " + viewName)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer sqldb.Query("drop view " + tmpViewName)
		modelName := namingStrategy.SchemaName(viewName)
		model := g.GenerateModelAs(tmpViewName, modelName)

		if ef, ok := extraFields[viewName]; ok {
			for _, ef := range ef {
				a := gen.FieldNew("", "", nil)
				f := a(nil)
				genRelationField(&ef, gen.Field(f))
				model.Fields = append(model.Fields, f)
			}
		}
		model.FileName = viewName
		model.TableName = viewName
		if jsonTagOverrides, ok := jsonTagOverridesByTable[viewName]; ok {
			for _, f := range model.Fields {
				if jsonTag, ok := jsonTagOverrides[f.ColumnName]; ok {
					f.Tag.Set("json", jsonTag)
				} else if jsonTag, ok := jsonTagOverrides[f.Name]; ok {
					f.Tag.Set("json", jsonTag)
				}
			}
		}
		modelsMap[viewName] = model
	}

	models := []any{}
	for _, model := range modelsMap {
		models = append(models, model)
	}
	g.ApplyBasic(models...)
	g.Execute()
}

func dt(goType string) func(columnType gorm.ColumnType) string {
	return func(columnType gorm.ColumnType) string { return goType }
}

func genRelationField(ef *ExtraField, fld gen.Field) {
	baseType := ef.StructPropType
	if lastDotIndex := strings.LastIndex(ef.StructPropType, "."); lastDotIndex != -1 {
		baseType = ef.StructPropType[lastDotIndex+1:]
	}
	if ef.Pointer {
		baseType = "*" + baseType
	}
	if ef.HasMany {
		baseType = "[]" + baseType
	}
	fld.Name = ef.StructPropName
	fld.Type = baseType
	t := field.Tag{}
	t.Set("json", strcase.ToLowerCamel(ef.StructPropName))
	fld.Tag = t
	fld.GORMTag = field.GormTag{}
	fld.GORMTag.Set("foreignKey", ef.FkStructPropName)
	fld.GORMTag.Set("references", ef.RefStructPropName)
	r := field.HasOne
	if ef.HasMany {
		r = field.HasMany
	}
	fld.Relation = field.NewRelationWithType(
		r,
		ef.StructPropName,
		ef.StructPropType,
	)
}

func cleanUp() {
	//Cleanup
	genFiles, err := filepath.Glob(outPath + "/*gen.go")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, genFile := range genFiles {
		os.Remove(genFile)
	}
	genFiles, err = filepath.Glob(modelPkgPath + "/*gen.go")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, genFile := range genFiles {
		os.Remove(genFile)
	}

}

func tableNames(db *gorm.DB) (tableNames []string) {
	tableNames = []string{}
	err := db.Raw("select table_name from information_schema.tables where table_schema = 'public'").Scan(&tableNames).Error
	if err != nil {
		log.Fatal(err.Error())
	}
	return
}

func materializedViewNames(db *gorm.DB) (tableNames []string) {
	tableNames = []string{}
	err := db.Raw("select matviewname from pg_matviews where schemaname='public'").Scan(&tableNames).Error
	if err != nil {
		log.Fatal(err.Error())
	}
	return
}
