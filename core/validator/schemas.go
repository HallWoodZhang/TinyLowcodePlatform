package validator

var CreateScriptSchema = Object(map[string]*Schema{
	"name":   String(),
	"label":  String(),
	"type":   String(),
	"tsCode": String(),
}, "name", "label")

var UpdateScriptSchema = Object(map[string]*Schema{
	"name":   String(),
	"label":  String(),
	"type":   String(),
	"tsCode": String(),
})

var DebugScriptSchema = Object(map[string]*Schema{
	"skip": Number(),
})

var SetBreakpointSchema = Object(map[string]*Schema{
	"line":    Integer(),
	"enabled": Boolean(),
}, "line", "enabled")

var RunSQLSchema = Object(map[string]*Schema{
	"sql": String(),
}, "sql")
