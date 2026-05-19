package docserver

import "fmt"

func scalarHTML(specURL, title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%s - API Reference</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <script>
    Scalar.createApiReference('#app', {
      spec: {
        url: '%s',
      },
    });
  </script>
</body>
</html>`, title, specURL)
}

func asyncAPIHTML(specURL, title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%s - Event Reference</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <link rel="stylesheet" href="https://unpkg.com/@asyncapi/react-component@latest/styles/default.min.css">
  <style>body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }</style>
</head>
<body>
  <div id="asyncapi"></div>
  <script src="https://unpkg.com/@asyncapi/react-component@latest/browser/standalone/index.js"></script>
  <script>
    AsyncApiStandalone.render({
      schema: {
        url: '%s',
        options: { method: "GET" },
      },
      config: {
        show: {
          sidebar: true,
          operations: true,
          messages: true,
          schemas: true,
        },
      },
    }, document.getElementById('asyncapi'));
  </script>
</body>
</html>`, title, specURL)
}
