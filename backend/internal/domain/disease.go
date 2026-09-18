package domain

type Disease struct {
	ID                string   `json:"id"`
	Plant             string   `json:"plant"`
	DiseaseName       string   `json:"disease_name"`
	DiseaseNameRU     string   `json:"disease_name_ru"`
	ScientificName    string   `json:"scientific_name"`
	Symptoms          []string `json:"symptoms"`
	Treatment         []string `json:"treatment"`
	ReferenceImageURL string   `json:"reference_image_url"`
	Source            string   `json:"source"`
	License           string   `json:"license,omitempty"`
}
