package main

import (
	"encoding/json"
	"net/http"
)

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":      "healthy",
		"environment": app.config.env,
		"version":     version,
	}
	if err := app.writeJSON(w, http.StatusOK, data, nil); err != nil {
		app.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// INFO http.Header is a map[string]string, but it's meaningful
func (app *application) writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for k, v := range headers {
		// INFO importannt. It allows you set all the headers
		w.Header()[k] = v
	}

	js = append(js, '\n')

	w.Header().Set("Content Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
