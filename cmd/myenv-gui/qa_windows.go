//go:build windows && gui && guiqa

package main

import (
	"context"
	"encoding/json"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"myenv/internal/cli"
	"myenv/internal/state"
	"os"
	"path/filepath"
	"time"
)

func (a *App) QAReport(message string) {
	f, err := os.OpenFile(os.Getenv("MYENV_GUI_QA_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer f.Close()
		f.WriteString(message + "\n")
	}
}
func (a *App) QAStartBlocked() (uint64, error) {
	directory := os.Getenv("MYENV_GUI_QA_PROJECT")
	unlock, err := state.LockWorkspace(context.Background(), filepath.Join(directory, ".myenv", "modify.lock"))
	if err != nil {
		return 0, err
	}
	id, err := a.Start(cli.UIRequest{Action: "sync", Directory: directory})
	go func() { defer unlock(); time.Sleep(3 * time.Second) }()
	return id, err
}
func init() {
	guiNamespace = os.Getenv("MYENV_GUI_QA_NAMESPACE")
	guiWebviewPath = os.Getenv("MYENV_GUI_QA_WEBVIEW")
	onGUIShutdown = func(t cli.UITask) {
		b, _ := json.Marshal(map[string]any{"shutdown": t})
		a := &App{}
		a.QAReport(string(b))
	}
	onGUIReady = func(ctx context.Context) {
		if path := os.Getenv("MYENV_GUI_QA_SCRIPT"); path != "" {
			if script, err := os.ReadFile(path); err == nil {
				runtime.WindowExecJS(ctx, string(script))
			}
			return
		}
		project, _ := json.Marshal(os.Getenv("MYENV_GUI_QA_PROJECT"))
		if os.Getenv("MYENV_GUI_QA_MODE") == "perf" {
			runtime.WindowExecJS(ctx, `setTimeout(async()=>{const a=window.go.main.App;await a.QAReport(JSON.stringify({ready:performance.now()}));setTimeout(async()=>{await a.Start({action:'doctor',deep:true,directory:`+string(project)+`})},15000);setTimeout(()=>window.runtime.Quit(),25000)},0)`)
			return
		}
		if args := os.Getenv("MYENV_GUI_QA_RUN"); args != "" {
			runtime.WindowExecJS(ctx, `setTimeout(async()=>{const a=window.go.main.App;try{await a.QAReport(JSON.stringify({handoff:await a.Run(`+string(project)+`,false,`+args+`)}))}catch(e){await a.QAReport(JSON.stringify({error:String(e)}))}window.runtime.Quit()},300)`)
			return
		}
		script := `setTimeout(async()=>{const app=window.go.main.App;try{const ready={text:document.body.innerText,ready:performance.now(),dpi:devicePixelRatio,width:innerWidth,height:innerHeight};const field=document.querySelector('input');const setter=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;setter.call(field,'中文 空格');field.dispatchEvent(new Event('input',{bubbles:true}));ready.input=field.value;await app.QAReport(JSON.stringify(ready));const id=await app.Start({action:'doctor',directory:PROJECT,deep:true});await new Promise((resolve,reject)=>{const stop=window.runtime.EventsOn('task',t=>{if(t.id===id&&!['running','confirm','canceling'].includes(t.state)){stop();app.QAReport(JSON.stringify({doctor:t}));resolve()}});setTimeout(()=>reject(Error('doctor timeout')),30000)});const blocked=await app.QAStartBlocked();await app.Cancel(blocked);window.runtime.Quit()}catch(e){app.QAReport(JSON.stringify({error:String(e)}))}},300);`
		script = replaceProject(script, string(project))
		runtime.WindowExecJS(ctx, script)
	}
}
func replaceProject(s, p string) string {
	for i := 0; i+7 <= len(s); i++ {
		if s[i:i+7] == "PROJECT" {
			return s[:i] + p + s[i+7:]
		}
	}
	return s
}
