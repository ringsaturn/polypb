package convert

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
	polypbv1 "github.com/ringsaturn/polypb/gen/polypb/v1"
)

const (
	MultiPolygonType = "MultiPolygon"
	PolygonType      = "Polygon"
	FeatureType      = "Feature"
)

type PolygonCoordinates [][][2]float64
type MultiPolygonCoordinates []PolygonCoordinates

type GeometryDefine struct {
	Coordinates interface{} `json:"coordinates"`
	Type        string      `json:"type"`
}

type PropertiesDefine struct {
	Tzid string `json:"tzid"`
}

type FeatureItem struct {
	Geometry   GeometryDefine   `json:"geometry"`
	Properties PropertiesDefine `json:"properties"`
	Type       string           `json:"type"`
}

type BoundaryFile struct {
	Type     string         `json:"type"`
	Features []*FeatureItem `json:"features"`
}

func Do(input *BoundaryFile) (*polypbv1.Shapes, error) {
	output := make([]*polypbv1.Shape, 0)

	for _, item := range input.Features {
		_data, err := json.Marshal(item.Properties)
		if err != nil {
			return nil, err
		}
		pbtzItem := &polypbv1.Shape{
			Data: _data,
		}

		var coordinates MultiPolygonCoordinates

		MultiPolygonTypeHandler := func() error {
			if err := mapstructure.Decode(item.Geometry.Coordinates, &coordinates); err != nil {
				return err
			}
			return nil
		}
		PolygonTypeHandler := func() error {
			var polygonCoordinates PolygonCoordinates
			if err := mapstructure.Decode(item.Geometry.Coordinates, &polygonCoordinates); err != nil {
				return err
			}
			coordinates = append(coordinates, polygonCoordinates)
			return nil
		}

		switch item.Type {
		case MultiPolygonType:
			if err := MultiPolygonTypeHandler(); err != nil {
				return nil, err
			}
		case PolygonType:
			if err := PolygonTypeHandler(); err != nil {
				return nil, err
			}
		case FeatureType:
			switch item.Geometry.Type {
			case MultiPolygonType:
				if err := MultiPolygonTypeHandler(); err != nil {
					return nil, err
				}
			case PolygonType:
				if err := PolygonTypeHandler(); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown type %v", item.Type)
			}
		default:
			return nil, fmt.Errorf("unknown type %v", item.Type)
		}

		polygons := make([]*polypbv1.Polygon, 0)

		for _, subcoordinates := range coordinates {
			newpbPoly := &polypbv1.Polygon{
				Points: make([]*polypbv1.Point, 0),
				Holes:  make([]*polypbv1.Polygon, 0),
			}
			for index, geoPoly := range subcoordinates {
				if index == 0 {
					for _, rawCoords := range geoPoly {
						newpbPoly.Points = append(newpbPoly.Points, &polypbv1.Point{
							Lng: float32(rawCoords[0]),
							Lat: float32(rawCoords[1]),
						})
					}
					continue
				}

				holePoly := &polypbv1.Polygon{
					Points: make([]*polypbv1.Point, 0),
				}
				for _, rawCoords := range geoPoly {
					holePoly.Points = append(holePoly.Points, &polypbv1.Point{
						Lng: float32(rawCoords[0]),
						Lat: float32(rawCoords[1]),
					})
				}
				newpbPoly.Holes = append(newpbPoly.Holes, holePoly)

			}
			polygons = append(polygons, newpbPoly)
		}

		pbtzItem.Polygons = polygons
		output = append(output, pbtzItem)
	}

	return &polypbv1.Shapes{
		Shapes: output,
	}, nil
}
