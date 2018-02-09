package forms

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/rumlang/rum/parser"
	"github.com/rumlang/rum/runtime"
	"github.com/satori/go.uuid"
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

func codeToLoad(ws *websocket.Conn) (string, error) {
	js := "ws.send(window.location.href);"

	err := ws.WriteMessage(websocket.TextMessage, []byte(js))
	if err != nil {
		return "", err
	}

	_, bret, err := ws.ReadMessage()
	if err != nil {
		return "", err
	}
	res := string(bret)

	if len(res) == 0 {
		return "", err
	}

	fs := strings.Split(res, "?")
	if len(fs) > 2 {
		return "", fmt.Errorf("Malformed URL")
	}

	if len(fs) == 2 {
		code, err := ioutil.ReadFile(fs[1] + ".rum")
		if err != nil {
			return "", err
		}
		return string(code), nil
	}

	return "", nil
}

//Render the initial html form to FormRumrepl
func (form RumReplForm) Render(ws *websocket.Conn, app *valente.App, params []string) error {
	content := elements.Panel{}
	content.AddClass("container")

	content.AddElement(elements.Heading3{Text: "The Rum Playground"})

	rowbtn := elements.Panel{}
	rowbtn.AddClass("row")

	btnRun := elements.Button{Text: "Run", PostBack: []string{"run"}}
	btnRun.AddClass("btn btn-primary col-3")

	btnShare := elements.Button{Text: "Share", PostBack: []string{"share"}}
	btnShare.AddClass("btn btn-info col-3 offset-1")

	btnClean := elements.Button{Text: "Clean", PostBack: []string{"clean"}}
	btnClean.AddClass("btn btn-secondary col-3 offset-1")

	rowbtn.AddElement(btnRun)
	rowbtn.AddElement(btnShare)
	rowbtn.AddElement(btnClean)
	content.AddElement(rowbtn)

	shareurl := elements.Panel{}
	shareurl.ID = "shareurl"
	shareurl.AddClass("row")
	shareurl.SetStyle("margin-top", "15px")
	content.AddElement(shareurl)

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

	action.Exec(ws, `
          $("#input").on("change keyup paste", function() {
             $("#shareurl").html("");
          });
        `)

	code, err := codeToLoad(ws)
	if err != nil {
		shareError(ws, fmt.Sprint("Error loading code", err))
	}

	err = action.Set(ws, "input", code)
	if err != nil {
		shareError(ws, fmt.Sprint("Error loading code", err))
	}

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
	action.HTML(ws, "shareurl", "")
	action.Set(ws, "input", "")
}

func shareError(ws *websocket.Conn, msg string) {
	log.Println(msg)
	action.HTML(ws, "shareurl", msg)
}

func shareRumRepl(ws *websocket.Conn, app *valente.App, params []string) {
	code, err := action.Get(ws, "input")
	if err != nil {
		shareError(ws, fmt.Sprint("Error getting code", err))
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		shareError(ws, fmt.Sprint("UUID", err))
	}

	err = ioutil.WriteFile(uuid.String()+".rum", []byte(code), 0644)
	if err != nil {
		shareError(ws, fmt.Sprint("Error saving code", err))
	}

	action.HTML(ws, "shareurl", `
             <div class="alert alert-success" role="alert">
	       <b>URL to share:</b> http://playground.rumlang.org/?`+uuid.String()+
		"</div>")
}

//Initialize inits the Rumrepl Form
func (form RumReplForm) Initialize(ws *websocket.Conn) valente.Form {
	log.Println("RumreplForm Initialize")

	f := form.AddEventHandler("run", runRumRepl)
	f = f.AddEventHandler("clean", cleanRumRepl)
	f = f.AddEventHandler("share", shareRumRepl)

	return f
}
