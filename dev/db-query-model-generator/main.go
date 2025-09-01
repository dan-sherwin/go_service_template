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
	db, err = gorm.Open(sqlite.Open("dev/db-query-model-generator/schema.db"), &gorm.Config{})
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

	// JSON tag strategy same as Postgres generator
	g.WithJSONTagNameStrategy(func(col string) (tag string) { return strcase.ToLowerCamel(col) })
	// Use SQLite-specific type map (no Postgres materialized views handling)
	g.WithDataTypeMap(sqliteTypeMap)
	g.WithImportPkgPath("gorm.io/datatypes")
	g.UseDB(db)

	// Build models to allow extraFields and jsonTagOverrides like Postgres path
	modelsMap := map[string]any{}
	for _, tableName := range sqliteTableNames(db) {
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

	models := []any{}
	for _, m := range modelsMap {
		models = append(models, m)
	}
	g.ApplyBasic(models...)
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

func sqliteTableNames(db *gorm.DB) (tableNames []string) {
	tableNames = []string{}
	err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableNames).Error
	if err != nil {
		log.Fatal(err.Error())
	}
	return
}

var sqliteTypeMap = map[string]func(gorm.ColumnType) string{
	// ---- booleans ----
	"BOOLEAN": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "bool")
	},
	"BOOL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "bool")
	},
	"TINYINT": func(ct gorm.ColumnType) string {
		// Treat TINYINT(1) as bool; otherwise int8
		col, _ := ct.ColumnType()
		n, _ := ct.Nullable()
		if strings.HasPrefix(strings.ToUpper(col), "TINYINT(1)") {
			return nullablePtr(n, "bool")
		}
		return nullablePtr(n, "int8")
	},

	// ---- integers ----
	"SMALLINT":         func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int16") },
	"INTEGER":          func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") }, // SQLite INTEGER is 64-bit
	"INT":              func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },
	"INT2":             func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int16") },
	"INT8":             func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },
	"MEDIUMINT":        func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int32") },
	"UNSIGNED BIG INT": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "uint64") },
	"BIGINT":           func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },

	// ---- floats ----
	"REAL":   func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float64") },
	"DOUBLE": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float64") },
	"FLOAT":  func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float32") },

	// ---- strings ----
	"TEXT":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"VARCHAR": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"CHAR":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"CLOB":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"UUID":    func(ct gorm.ColumnType) string { return "datatypes.UUID" },
	"JSON":    func(ct gorm.ColumnType) string { return "datatypes.JSON" },
	"JSONB":   func(ct gorm.ColumnType) string { return "datatypes.JSON" },

	// ---- bytes ----
	"BLOB": func(gorm.ColumnType) string { return "[]byte" },

	// ---- dates/times ----
	"DATE": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	"DATETIME": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	"TIMESTAMP": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	// Duration-like types (custom schemas often use these as declared types)
	"DURATION": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Duration")
	},
	"INTERVAL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Duration")
	},

	// ---- decimals / numerics ----
	// If you need exact decimals, map to shopspring/decimal and import it.
	"NUMERIC": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "float64")
	},
	"DECIMAL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "float64")
	},
}

func nullablePtr(yes bool, base string) string {
	if yes {
		// make a pointer for nullable scalar types
		switch base {
		case "bool", "int", "int8", "int16", "int32", "int64", "uint64", "float32", "float64", "string", "time.Time", "time.Duration":
			return "*" + base
		}
	}
	return base
}
