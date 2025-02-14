package main

type Takeout struct {
	Title                 string             `json:"title"`
	Description           string             `json:"description"`
	ImageViews            string             `json:"imageViews"`
	CreationTime          Time               `json:"creationTime"`
	PhotoTakenTime        Time               `json:"photoTakenTime"`
	GeoData               GeoData            `json:"geoData"`
	GeoDataExif           GeoData            `json:"geoDataExif"`
	URL                   string             `json:"url"`
	GooglePhotosOrigin    GooglePhotosOrigin `json:"googlePhotosOrigin"`
	PhotoLastModifiedTime Time               `json:"photoLastModifiedTime"`
}

type Time struct {
	Timestamp string `json:"timestamp"`
	Formatted string `json:"formatted"`
}

type GeoData struct {
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Altitude      float64 `json:"altitude"`
	LatitudeSpan  float64 `json:"latitudeSpan"`
	LongitudeSpan float64 `json:"longitudeSpan"`
}

type GooglePhotosOrigin struct {
	MobileUpload MobileUpload `json:"mobileUpload"`
}

type MobileUpload struct {
	DeviceType string `json:"deviceType"`
}
