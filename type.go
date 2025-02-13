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
	Latitude      int64 `json:"latitude"`
	Longitude     int64 `json:"longitude"`
	Altitude      int64 `json:"altitude"`
	LatitudeSpan  int64 `json:"latitudeSpan"`
	LongitudeSpan int64 `json:"longitudeSpan"`
}

type GooglePhotosOrigin struct {
	MobileUpload MobileUpload `json:"mobileUpload"`
}

type MobileUpload struct {
	DeviceType string `json:"deviceType"`
}
