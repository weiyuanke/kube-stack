package debugapi

import (
	"net/http"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	tpl = `<!DOCTYPE html>
<html lang="en">
 <head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta http-equiv="X-UA-Compatible" content="ie=edge" />
  <title>Document</title>
  <link href="/statics/reset.css" rel="stylesheet" />
  <link href="/statics/app.css" rel="stylesheet" />
  <link href="/statics/jsonTree.css" rel="stylesheet" />
  <link href="/statics/css_family.css" rel="stylesheet" />
 </head>
 <body>
  <header id="header">
   <nav id="nav" class="clearfix">
    <ul class="menu menu_level1">
     <li data-action="expand" class="menu__item" id="expand-all"> <span class="menu__item-name" style="color:black">展开内容>>></span> </li>
     <li data-action="collapse" class="menu__item" id="collapse-all"> <span class="menu__item-name" style="color:black">折叠内容>>></span> </li>
    </ul>
   </nav>
   <div id="coords"></div>
   <div id="debug"></div>
  </header>
  <div id="wrapper">
   <div id="tree"></div>
  </div>
  <script src="/statics/jsonTree.js"></script>
  <script>
	// Get DOM-element for inserting json-tree
	var wrapper = document.getElementById("wrapper");
	var data = %s
	var tree = jsonTree.create(data, wrapper);
	document.getElementById("expand-all").addEventListener("click", function() {
  		tree.expand();
	})
	document.getElementById("collapse-all").addEventListener("click", function() {
  		tree.collapse();
	})
  </script>
 </body>
</html>
`
)

type processer interface {
	Process(r *http.Request) (string, error)
}

func handlerWrapper(p processer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		jsonOption := r.URL.Query().Get("json")

		dataStr, err := p.Process(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		var respBody []byte
		if jsonOption == "true" {
			respBody = []byte(dataStr)
		} else {
			respBody = []byte(strings.Replace(tpl, "%s", strings.TrimSpace(dataStr), -1))
		}

		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)
		len := len(respBody)
		if len > 0 {
			w.Header().Set("Content-Length", strconv.Itoa(len))
			w.Write(respBody)
		}
	})
}

// RegisteToServer Registe handler for debug api
func RegisteToServer(server *webhook.Server) {
	server.Register("/podyaml", handlerWrapper(&podYamlProcessor{}))
	server.Register("/nodeyaml", handlerWrapper(&nodeYamlProcessor{}))
	server.Register("/eventyaml", handlerWrapper(&eventYamlProcessor{}))
	server.Register("/debugpod", handlerWrapper(&debugPodProcessor{}))
}
