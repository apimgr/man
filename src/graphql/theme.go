// Package graphql provides GraphQL API for casman.
// See AI.md for details.
package graphql

// DarkThemeCSS provides dark theme styling for GraphiQL.
const DarkThemeCSS = `
.graphiql-container.theme-dark {
  background: #282a36;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .CodeMirror {
  background: #282a36;
  color: #f8f8f2;
}

.graphiql-container.theme-dark .CodeMirror-gutters {
  background: #1e1f29;
  border-right: 1px solid #44475a;
}

.graphiql-container.theme-dark .result-window {
  background: #282a36;
}

.graphiql-container.theme-dark .execute-button {
  background: #50fa7b;
  color: #282a36;
}

.graphiql-container.theme-dark .toolbar-button {
  background: #44475a;
  color: #f8f8f2;
}
`

// LightThemeCSS provides light theme styling for GraphiQL.
const LightThemeCSS = `
.graphiql-container.theme-light {
  background: #ffffff;
  color: #1a1a1a;
}

.graphiql-container.theme-light .CodeMirror {
  background: #ffffff;
  color: #1a1a1a;
}

.graphiql-container.theme-light .CodeMirror-gutters {
  background: #f5f5f5;
  border-right: 1px solid #e0e0e0;
}

.graphiql-container.theme-light .result-window {
  background: #ffffff;
}

.graphiql-container.theme-light .execute-button {
  background: #008000;
  color: #ffffff;
}

.graphiql-container.theme-light .toolbar-button {
  background: #f5f5f5;
  color: #1a1a1a;
  border: 1px solid #cccccc;
}
`
