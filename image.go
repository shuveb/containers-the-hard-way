package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type manifest []struct {
	Config string
	RepoTags []string
	Layers []string
}

/*
{
	"ubuntu" : {
					"18.04": "/path/to/image/home",
					"18.10": "/path/to/image/home",
					"19.04": "/path/to/image/home",
					"19.10": "/path/to/image/home",
				},
	"centos" : {
					"6.0": "/path/to/image/home",
					"6.1": "/path/to/image/home",
					"6.2": "/path/to/image/home",
					"7.0": "/path/to/image/home",
				}
}
*/

type imageEntries map[string]string
type imagesDB map[string]imageEntries

func getBasePathForImage(imageShaHex string) string {
	return getGockerImagesPath() + "/" + imageShaHex
}

func getManifestPathForImage(imageShaHex string) string {
	return getBasePathForImage(imageShaHex) + "/manifest.json"
}

func deleteTempImageFiles(imageShaHash string) {
	tmpPath := getGockerTempPath() + "/" + imageShaHash
	doOrDieWithMsg(os.RemoveAll(tmpPath),
		"Unable to remove temporary image files")
}

func getImageAndTagForHash(imageShaHash string) (string, string) {
	idb := imagesDB{}
	parseImagesMetadata(&idb)
	for image,versions := range idb {
		for version, hash := range versions {
			if hash == imageShaHash {
				return image, version
			}
		}
	}
	return "", ""
}

func imageExistsByHash(imageShaHex string) (string, string) {
	idb := imagesDB{}
	parseImagesMetadata(&idb)
	for imgName, avlImages := range idb {
		for imgTag, imgHash := range avlImages {
			if imgHash == imageShaHex {
				return imgName, imgTag
			}
		}
	}
	return "", ""
}
func imageExistByTag(imgName string, tagName string) (bool, string) {
	idb := imagesDB{}
	parseImagesMetadata(&idb)
	for k, v := range idb {
		if k == imgName {
			for k, v := range v {
				if k == tagName {
					return true, v
				}
			}
		}
	}
	return false, ""
}

func downloadImage(img v1.Image, imageShaHex string, src string) {
	path := getGockerTempPath() + "/" + imageShaHex
	os.Mkdir(path, 0755)
	path +="/package.tar"
	/* Save the image as a tar file */
	if err := crane.SaveLegacy(img, src, path); err != nil {
		log.Fatalf("saving tarball %s: %v", path, err)
	}
	log.Printf("Successfully downloaded %s\n", src)
}

func untarFile(imageShaHex string) {
	pathDir := getGockerTempPath() + "/" + imageShaHex
	pathTar := pathDir + "/package.tar"
	if err := untar(pathTar, pathDir); err != nil {
		log.Fatalf("Error untaring file: %v\n", err)
	}
}

func processLayerTarballs(imageShaHex string) {
	tmpPathDir := getGockerTempPath() + "/" + imageShaHex
	pathManifest := tmpPathDir + "/manifest.json"

	mani := manifest{}
	parseManifest(pathManifest, &mani)
	if len(mani) == 0 || len(mani[0].Layers) == 0 {
		log.Fatal("Could not find any layers.")
	}
	if len(mani) > 1 {
		log.Fatal("I don't know how to handle more than one manifest.")
	}

	imagesDir := getGockerImagesPath() + "/" + imageShaHex
	_ = os.Mkdir(imagesDir, 0755)
	/* untar the layer files. These become the basis of our container root fs */
	for _, layer := range mani[0].Layers {
		imageLayerDir := imagesDir + "/" + layer[:12] + "/fs"
		log.Printf("Uncompressing layer to: %s \n", imageLayerDir)
		_ = os.MkdirAll(imageLayerDir, 0755)
		srcLayer := tmpPathDir + "/" + layer
		if err:= untar(srcLayer, imageLayerDir); err != nil {
			log.Fatalf("Unable to untar layer file: %s: %v\n", srcLayer, err)
		}
	}
	/* Copy the manifest file for reference later */
	copyFile(pathManifest, getManifestPathForImage(imageShaHex))
}

func parseImagesMetadata(idb *imagesDB)  {
	imagesDBPath := getGockerImagesPath() + "/" + "images.json"
	if _, err := os.Stat(imagesDBPath); os.IsNotExist(err) {
		/* If it doesn't exist create an empty DB */
		ioutil.WriteFile(imagesDBPath, []byte("{}"), 0644)
	}
	data, err := ioutil.ReadFile(imagesDBPath)
	if err != nil {
		log.Fatalf("Could not read images DB: %v\n", err)
	}
	if err := json.Unmarshal(data, idb); err != nil {
		log.Fatalf("Unable to parse images DB: %v\n", err)
	}
}

func storeImageMetadata(image string, tag string, imageShaHex string) {
	idb := imagesDB{}
	ientry := imageEntries{}
	parseImagesMetadata(&idb)
	log.Printf("Images Metadata: %v\n", idb)
	if idb[image] != nil {
		ientry = idb[image]
	}
	ientry[tag] = imageShaHex
	idb[image] = ientry

	fileBytes, err := json.Marshal(idb)
	if err != nil {
		log.Fatalf("Unable to marshall images data: %v\n", err)
	}
	imagesDBPath := getGockerImagesPath() + "/" + "images.json"
	if err := ioutil.WriteFile(imagesDBPath, fileBytes, 0644); err != nil {
		log.Fatalf("Unable to save images DB: %v\n", err)
	}
}

func printAvailableImages() {
	idb := imagesDB{}
	parseImagesMetadata(&idb)
	fmt.Printf("IMAGE\t             TAG\t   ID\n")
	for image, details := range idb {
		fmt.Println(image)
		for tag, hash := range details {
			fmt.Printf("\t%16s %s\n", tag, hash)
		}
	}
}

func getImageNameAndTag(src string) (string, string) {
	s := strings.Split(src, ":")
	var img, tag string
	if len(s) > 1 {
		img, tag = s[0], s[1]
	} else {
		img = s[0]
		tag = "latest"
	}
	return img, tag
}

func downloadImageIfRequired(src string) string {
	imgName, tagName := getImageNameAndTag(src)
	if downloadRequired, imageShaHex := imageExistByTag(imgName, tagName); !downloadRequired {
		/* Setup the image we want to pull */
		log.Printf("Downloading metadata for %s:%s, please wait...", imgName, tagName)
		img, err := crane.Pull(strings.Join([]string{imgName, tagName}, ":"))
		if err != nil {
			log.Fatal(err)
		}

		manifest, _ := img.Manifest()
		imageShaHex = manifest.Config.Digest.Hex[:12]
		log.Printf("imageHash: %v\n", imageShaHex)
		log.Println("Checking if image exists under another name...")
		/* Identify cases where ubuntu:latest could be the same as ubuntu:20.04*/
		altImgName, altImgTag := imageExistsByHash(imageShaHex)
		if len(altImgName) > 0 && len(altImgTag) > 0 {
			log.Printf("The image you requested %s:%s is the same as %s:%s\n",
				imgName, tagName, altImgName, altImgTag)
			storeImageMetadata(imgName, tagName, imageShaHex)
			return imageShaHex
		} else {
			log.Println("Image doesn't exist. Downloading...")
			downloadImage(img, imageShaHex, src)
			untarFile(imageShaHex)
			processLayerTarballs(imageShaHex)
			storeImageMetadata(imgName, tagName, imageShaHex)
			deleteTempImageFiles(imageShaHex)
			return imageShaHex
		}
	} else {
		log.Println("Image already exists. Not downloading.")
		return imageShaHex
	}
}

