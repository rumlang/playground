package forms

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/rumlang/rum/parser"
	"github.com/rumlang/rum/runtime"
	"github.com/trumae/valente"
	"github.com/trumae/valente/action"
	"github.com/trumae/valente/elements"
)

func rumParse(s string) (*parser.Value, error) {
	v, err := parser.Parse(parser.NewSource(s))
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func rumEval(s string, c *runtime.Context) (*parser.Value, error) {
	ival, err := rumParse(s)
	if err != nil {
		return nil, err
	}

	val, err := c.TryEval(*ival)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

//RumReplForm struct
type RumReplForm struct {
	valente.FormImpl
}

//Render the initial html form to FormRumrepl
func (form RumReplForm) Render(ws *websocket.Conn, app *valente.App, params []string) error {
	content := elements.Panel{}
	content.AddClass("container")

	content.AddElement(elements.Heading3{Text: "The Rum Playground"})

	rowbtn := elements.Panel{}
	rowbtn.AddClass("row")

	btnRun := elements.Button{Text: "Run", PostBack: []string{"run"}}
	btnRun.AddClass("btn btn-primary col-4")

	btnClean := elements.Button{Text: "Clean", PostBack: []string{"clean"}}
	btnClean.AddClass("btn btn-secondary col-4 offset-4")

	rowbtn.AddElement(btnRun)
	rowbtn.AddElement(btnClean)
	content.AddElement(rowbtn)

	codeta := elements.TextArea{}
	codeta.ID = "input"
	codeta.AddClass("form-control")
	codeta.SetData("rows", "20")
	codeta.SetData("cols", "80")
	codeta.SetStyle("height", "300px")
	codeta.SetStyle("width", "100%")
	codeta.SetStyle("margin-top", "10px")
	content.AddElement(elements.Heading4{Text: "Input"})
	content.AddElement(codeta)

	output := elements.Panel{}
	output.ID = "output"
	output.SetStyle("height", "300px")
	output.SetStyle("width", "100%")
	output.SetStyle("background-color", "#EEEEEE")
	output.SetStyle("overflow", "auto")
	content.AddElement(elements.Heading4{Text: "Output"})
	content.AddElement(output)

	action.HTML(ws, "content", content.String())
	return nil
}

func runRumRepl(ws *websocket.Conn, app *valente.App, params []string) {
	context, ok := app.Data["runContext"]
	if !ok {
		context = runtime.NewContext(nil)
		app.Data["runContext"] = context
	}
	code, err := action.Get(ws, "input")
	if err != nil {
		log.Println(err)
	}

	if len(code) > 0 {
		action.Append(ws, "output", "<p> >> <b>"+code+"</b><p>")

		v, err := rumEval(code, context.(*runtime.Context))
		if err != nil {
			action.Append(ws, "output", fmt.Sprint("<p>Err: ", err, "</p><br/>"))
		}

		action.Append(ws, "output", "<p>-> "+fmt.Sprintf("%v</p>", (*v).Value()))

		action.Exec(ws, "$('#output').scrollTop($('#output')[0].scrollHeight)")
	}
}

func cleanRumRepl(ws *websocket.Conn, app *valente.App, params []string) {
	action.HTML(ws, "output", "")
	action.Set(ws, "input", "")
}

//Initialize inits the Rumrepl Form
func (form RumReplForm) Initialize(ws *websocket.Conn) valente.Form {
	log.Println("RumreplForm Initialize")

	f := form.AddEventHandler("run", runRumRepl)
	f = f.AddEventHandler("clean", cleanRumRepl)

	return f
}
