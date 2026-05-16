package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteAlertRedirect responds with a tiny HTML page that pops a browser alert
// and then sends the user back to the previous page (Referer) — falling back
// to fallbackPath if no usable Referer is set or it points off-origin.
//
// Use this for non-fatal errors that would otherwise replace the whole page
// (permission denied, validation failures, conflicting deletes). The status
// code is set by the caller.
func WriteAlertRedirect(w http.ResponseWriter, r *http.Request, status int, message, fallbackPath string) {
	if fallbackPath == "" {
		fallbackPath = "/dashboard"
	}
	msgJSON, _ := json.Marshal(message)
	fallbackJSON, _ := json.Marshal(fallbackPath)
	referer := r.Referer()
	refJSON, _ := json.Marshal(referer)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="zh-Hant"><head><meta charset="utf-8"><title>%s</title></head>
<body><script>
(function(){
  var msg = %s, ref = %s, fallback = %s;
  alert(msg);
  try {
    if (ref && new URL(ref).origin === location.origin && ref !== location.href) {
      location.replace(ref);
      return;
    }
  } catch (e) {}
  location.replace(fallback);
})();
</script></body></html>`, message, msgJSON, refJSON, fallbackJSON)
}
