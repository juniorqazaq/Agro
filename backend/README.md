# Plantix backend MVP

Go API that sends a plant image to Gemini, receives only a `disease_id`, and
returns trusted disease details from the local JSON knowledge base.

## Run

Requires Go 1.25.5 or newer.

```powershell
Copy-Item .env.example .env
# Add GEMINI_API_KEY to .env
go run ./cmd/server
```

For frontend development without an API key, set this in `.env`:

```dotenv
AI_MOCK_DISEASE_ID=tomato_early_blight
```

Server: `http://localhost:8000`. Health check: `GET /api/health`.

## Analyze endpoint

`POST /api/analyze` accepts `multipart/form-data` with an image in the `file`
field. Supported formats are JPEG, PNG, and WebP, up to 10 MB.

```powershell
curl.exe -X POST http://localhost:8000/api/analyze `
  -F "file=@testdeseasePhotos/leaf.jpg"
```

Known disease (`200`):

```json
{
  "success": true,
  "disease": {
    "id": "tomato_early_blight",
    "plant": "Tomato",
    "disease_name": "Early Blight",
    "disease_name_ru": "Альтернариоз (ранняя сухая пятнистость)",
    "scientific_name": "Alternaria solani",
    "symptoms": ["..."],
    "treatment": ["..."],
    "reference_image_url": "https://plantvillage.org",
    "source": "PlantVillage Dataset",
    "license": "CC BY-NC-SA 4.0"
  }
}
```

Not recognized (`200`):

```json
{
  "success": true,
  "disease": null,
  "message": "Disease could not be recognized from this image."
}
```

Invalid request (`400`, `413`, or `415`):

```json
{
  "success": false,
  "error": {
    "code": "INVALID_FILE_TYPE",
    "message": "Only JPEG, PNG, or WebP images are supported."
  }
}
```

The frontend origin `http://localhost:5173` is allowed by CORS. To add diseases,
Mansur can replace or extend `data/disease_database.json`; every `id` must be
unique. Treatment entries are the recommendations shown to the user.
