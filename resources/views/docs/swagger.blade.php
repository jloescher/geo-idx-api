<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Quantyra GeoIDX API Docs (Swagger UI)</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html {
            box-sizing: border-box;
            overflow-y: scroll;
        }

        *,
        *::before,
        *::after {
            box-sizing: inherit;
        }

        body {
            margin: 0;
            background: #fafafa;
        }
    </style>
</head>
<body>
<div id="swagger-ui"></div>

<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
<script>
    window.addEventListener('load', function () {
        window.SwaggerUIBundle({
            url: @json($openApiSpecUrl),
            dom_id: '#swagger-ui',
            deepLinking: true,
            displayRequestDuration: true,
            tryItOutEnabled: true,
            docExpansion: 'list',
            persistAuthorization: true,
        });
    });
</script>
</body>
</html>
