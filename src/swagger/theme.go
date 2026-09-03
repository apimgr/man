// Package swagger provides OpenAPI/Swagger specification and UI for casman.
// See AI.md for details.
package swagger

// DarkThemeCSS provides dark theme styling for Swagger UI.
const DarkThemeCSS = `
.swagger-ui.theme-dark {
  background: #282a36;
  color: #f8f8f2;
}

.swagger-ui.theme-dark .topbar {
  background: #1e1f29;
}

.swagger-ui.theme-dark .info .title,
.swagger-ui.theme-dark .opblock-tag {
  color: #f8f8f2;
}

.swagger-ui.theme-dark .opblock.opblock-get {
  background: rgba(139, 233, 253, 0.1);
  border-color: #8be9fd;
}

.swagger-ui.theme-dark .opblock.opblock-post {
  background: rgba(80, 250, 123, 0.1);
  border-color: #50fa7b;
}

.swagger-ui.theme-dark .opblock.opblock-put {
  background: rgba(255, 184, 108, 0.1);
  border-color: #ffb86c;
}

.swagger-ui.theme-dark .opblock.opblock-delete {
  background: rgba(255, 85, 85, 0.1);
  border-color: #ff5555;
}

.swagger-ui.theme-dark input,
.swagger-ui.theme-dark textarea,
.swagger-ui.theme-dark select {
  background: #44475a;
  color: #f8f8f2;
  border: 1px solid #6272a4;
}

.swagger-ui.theme-dark .btn {
  background: #6272a4;
  color: #f8f8f2;
}
`

// LightThemeCSS provides light theme styling for Swagger UI.
const LightThemeCSS = `
.swagger-ui.theme-light {
  background: #ffffff;
  color: #1a1a1a;
}

.swagger-ui.theme-light .topbar {
  background: #f5f5f5;
  border-bottom: 1px solid #e0e0e0;
}

.swagger-ui.theme-light .info .title,
.swagger-ui.theme-light .opblock-tag {
  color: #1a1a1a;
}

.swagger-ui.theme-light .opblock.opblock-get {
  background: rgba(0, 102, 204, 0.05);
  border-color: #0066cc;
}

.swagger-ui.theme-light .opblock.opblock-post {
  background: rgba(0, 128, 0, 0.05);
  border-color: #008000;
}

.swagger-ui.theme-light .opblock.opblock-put {
  background: rgba(255, 140, 0, 0.05);
  border-color: #ff8c00;
}

.swagger-ui.theme-light .opblock.opblock-delete {
  background: rgba(204, 0, 0, 0.05);
  border-color: #cc0000;
}

.swagger-ui.theme-light input,
.swagger-ui.theme-light textarea,
.swagger-ui.theme-light select {
  background: #ffffff;
  color: #1a1a1a;
  border: 1px solid #cccccc;
}

.swagger-ui.theme-light .btn {
  background: #0066cc;
  color: #ffffff;
}
`
