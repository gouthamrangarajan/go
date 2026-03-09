package models

type DemoItem struct {
	Id                int
	Title             string
	ImgSrc            string
	Description       string
	Url               string
	Service           string
	Tags              string
	ImgBadgeLightMode bool
	CodeUrl           string
	Display           bool
	Embeddings        []float32
}
