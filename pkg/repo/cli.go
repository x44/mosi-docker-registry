package repo

import (
	"fmt"
	"mosi-docker-registry/pkg/filesys"
	"mosi-docker-registry/pkg/json"
	"mosi-docker-registry/pkg/wildcard"
)

func List(imgPattern, tagPattern string) (*json.JsonObject, error) {
	if tagPattern == "" {
		return listImages(imgPattern)
	} else {
		return listLayers(imgPattern, tagPattern)
	}
}

func listImages(imgPattern string) (*json.JsonObject, error) {
	imgs, err := getImages()
	if err != nil {
		return nil, err
	}

	tables := json.NewJsonArray(0)
	res := json.NewJsonObject()
	res.Put("tables", tables)

	var table *json.JsonObject = nil
	var rows *json.JsonArray = nil

	for _, img := range imgs {
		if wildcard.Matches(img, imgPattern) {

			if table == nil {
				table = json.NewJsonObject()
				table.Put("fields", json.JsonArrayFromStrings("Image", "Tags", "Blobs", "Size"))
				tables.Add(table)

				rows = json.NewJsonArray(0)
				table.Put("rows", rows)
			}

			nTags := -1
			tags, err := getImageTags(img)
			if err != nil {
				return nil, err
			}
			nTags = len(tags)

			nBlobs := -1
			var nBlobBytes int64 = 0
			blobs, err := getBlobFiles(img)
			if err != nil {
				return nil, err
			}
			nBlobs = len(blobs)

			for _, blob := range blobs {
				size, err := filesys.Size(blob)
				if err != nil {
					return nil, err
				}
				nBlobBytes += size
			}
			rows.Add(json.JsonArrayFromAny(img, nTags, nBlobs, filesys.Bytes2IEC(nBlobBytes)))
		}
	}
	return res, nil
}

func listLayers(imgPattern, tagPattern string) (*json.JsonObject, error) {
	imgs, err := getImages()
	if err != nil {
		return nil, err
	}

	tables := json.NewJsonArray(0)
	res := json.NewJsonObject()
	res.Put("tables", tables)

	for _, img := range imgs {
		if wildcard.Matches(img, imgPattern) {

			tags, err := getImageTags(img)
			if err != nil {
				return nil, err
			}
			for _, tag := range tags {
				if wildcard.Matches(tag, tagPattern) {

					table := json.NewJsonObject()
					tables.Add(table)
					table.Put("fields", json.JsonArrayFromStrings("Image", "Tag", "Layer", "Size"))
					rows := json.NewJsonArray(0)
					table.Put("rows", rows)

					manifestJson, err := getManifestJson(img, tag)
					if err != nil {
						return nil, err
					}

					layerDigests, err := getManifestLayerDigests(manifestJson)
					if err != nil {
						return nil, err
					}

					for _, layerDigest := range layerDigests {
						servedBlobFn, err := getBlobServedFilename(img, layerDigest)
						if err != nil {
							return nil, err
						}
						nLayerBytes, err := filesys.Size(servedBlobFn)
						if err != nil {
							return nil, err
						}

						rows.Add(json.JsonArrayFromStrings(img, tag, layerDigest, filesys.Bytes2IEC(nLayerBytes)))
					}
				}
			}
		}
	}
	return res, nil
}

func Delete(imgPattern, tagPattern string, dry bool) (*json.JsonObject, error) {
	if tagPattern == "" {
		tagPattern = "*"
	}
	return deleteImages(imgPattern, tagPattern, dry)
}

func deleteImages(imgPattern, tagPattern string, dry bool) (*json.JsonObject, error) {
	imgs, err := getImages()
	if err != nil {
		return nil, err
	}

	tables := json.NewJsonArray(0)
	res := json.NewJsonObject()
	res.Put("tables", tables)

	var table *json.JsonObject = nil
	var rows *json.JsonArray = nil

	imgsDeleted := make(map[string]bool)

	for _, img := range imgs {
		if wildcard.Matches(img, imgPattern) {

			tags, err := getImageTags(img)
			if err != nil {
				return nil, err
			}
			for _, tag := range tags {
				if wildcard.Matches(tag, tagPattern) {

					imgsDeleted[img] = true

					if table == nil {
						table = json.NewJsonObject()
						table.Put("fields", json.JsonArrayFromStrings("Image", "Tag", "Deleted"))
						tables.Add(table)

						rows = json.NewJsonArray(0)
						table.Put("rows", rows)
					}

					s := "NO"
					if !dry {
						s = "YES"

						err = deleteImage(img, tag)
						if err != nil {
							s = fmt.Sprintf("NO, ERROR: %v", err)
						}
					}
					rows.Add(json.JsonArrayFromStrings(img, tag, s))
				}
			}
		}
	}

	if !dry {
		for img := range imgsDeleted {
			CleanupImage(img)
		}
	}

	return res, nil
}
