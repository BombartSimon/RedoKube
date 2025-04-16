package mockers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"gopkg.in/yaml.v3"
)

// MockOpenAPISpec adds fake examples to an OpenAPI specification
func MockOpenAPISpec(specContent string) (string, error) {
	var swagger map[string]interface{}

	// Try parsing as JSON first
	err := json.Unmarshal([]byte(specContent), &swagger)
	if err != nil {
		// If JSON parsing fails, try YAML
		err = yaml.Unmarshal([]byte(specContent), &swagger)
		if err != nil {
			return "", fmt.Errorf("error parsing OpenAPI spec (neither valid JSON nor YAML): %v", err)
		}
	}

	log.Printf("Parsed swagger: %+v", swagger)

	gofakeit.Seed(0)

	// Create OpenAPI 3.0 skeleton if needed
	openapi := make(map[string]interface{})

	// Check if already OpenAPI 3.0+
	if version, ok := swagger["openapi"].(string); ok && strings.HasPrefix(version, "3.") {
		openapi = swagger
	} else {
		// Convert from Swagger 2.0 to OpenAPI 3.0
		openapi["openapi"] = "3.1.0"
		openapi["info"] = swagger["info"]

		// Add servers if host exists
		if host, ok := swagger["host"].(string); ok {
			basePath := "/"
			if bp, ok := swagger["basePath"].(string); ok {
				basePath = bp
			}
			openapi["servers"] = []map[string]string{
				{"url": fmt.Sprintf("https://%s%s", host, basePath)},
			}
		}

		// Convert definitions to components/schemas
		if defsRaw, ok := swagger["definitions"].(map[string]interface{}); ok {
			openapi["components"] = map[string]interface{}{
				"schemas": defsRaw,
			}
		}
	}

	// Get definitions/schemas
	var schemas map[string]interface{}
	if components, ok := openapi["components"].(map[string]interface{}); ok {
		if s, ok := components["schemas"].(map[string]interface{}); ok {
			schemas = s
		}
	} else if defsRaw, ok := swagger["definitions"].(map[string]interface{}); ok {
		schemas = defsRaw
	}

	// Process paths
	paths := make(map[string]interface{})
	var rawPaths map[string]interface{}

	if p, ok := swagger["paths"].(map[string]interface{}); ok {
		rawPaths = p
	} else if p, ok := openapi["paths"].(map[string]interface{}); ok {
		rawPaths = p
	}

	if rawPaths == nil {
		log.Println("Warning: No paths found in the OpenAPI spec")
		rawPaths = make(map[string]interface{})
	}

	for pathKey, pathVal := range rawPaths {
		pathItem, ok := pathVal.(map[string]interface{})
		if !ok {
			log.Printf("Warning: Path item is not a map: %v", pathVal)
			continue
		}

		newPathItem := make(map[string]interface{})

		for methodKey, methodVal := range pathItem {
			method, ok := methodVal.(map[string]interface{})
			if !ok {
				log.Printf("Warning: Method is not a map: %v", methodVal)
				continue
			}

			newMethod := make(map[string]interface{})

			// Copy description, summary, etc.
			for k, v := range method {
				if k != "responses" {
					newMethod[k] = v
				}
			}

			responses := make(map[string]interface{})
			if resp, ok := method["responses"].(map[string]interface{}); ok {
				for statusCode, respVal := range resp {
					response, ok := respVal.(map[string]interface{})
					if !ok {
						log.Printf("Warning: Response is not a map: %v", respVal)
						continue
					}

					newResp := make(map[string]interface{})
					newResp["description"] = response["description"]

					// Add examples for successful responses
					if statusCode == "200" || statusCode == "201" {
						if schema, ok := response["schema"].(map[string]interface{}); ok {
							if ref, ok := schema["$ref"].(string); ok {
								refName := resolveRef(ref)
								if def, ok := schemas[refName]; ok {
									defMap, ok := def.(map[string]interface{})
									if !ok {
										log.Printf("Warning: Schema definition is not a map: %v", def)
										continue
									}

									example := generateFakeObject(defMap, schemas)
									newResp["content"] = map[string]interface{}{
										"application/json": map[string]interface{}{
											"schema": map[string]interface{}{
												"$ref": convertRefPath(ref),
											},
											"examples": map[string]interface{}{
												"auto_example": map[string]interface{}{
													"value": example,
												},
											},
										},
									}
								}
							}
						}
					} else {
						// Generate simple example for errors
						example := map[string]interface{}{
							"message":   gofakeit.Sentence(5),
							"errorCode": gofakeit.Regex(`ERR_[0-9]{3}`),
						}
						newResp["content"] = map[string]interface{}{
							"application/json": map[string]interface{}{
								"examples": map[string]interface{}{
									"auto_example": map[string]interface{}{
										"value": example,
									},
								},
							},
						}
					}

					responses[statusCode] = newResp
				}
			}

			newMethod["responses"] = responses
			newPathItem[methodKey] = newMethod
		}
		paths[pathKey] = newPathItem
	}
	openapi["paths"] = paths

	// Convert back to YAML
	out, err := yaml.Marshal(openapi)
	if err != nil {
		return "", fmt.Errorf("error encoding modified OpenAPI spec: %v", err)
	}

	log.Println("Successfully generated OpenAPI examples")
	return string(out), nil
}

// Helper function to extract the reference name
func resolveRef(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// Convert Swagger 2.0 ref path to OpenAPI 3.0
func convertRefPath(ref string) string {
	if strings.HasPrefix(ref, "#/definitions/") {
		return strings.Replace(ref, "#/definitions/", "#/components/schemas/", 1)
	}
	return ref
}

// Generate fake object based on schema definition
func generateFakeObject(def map[string]interface{}, defs map[string]interface{}) map[string]interface{} {
	props, ok := def["properties"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{"example": "No properties found"}
	}

	result := make(map[string]interface{})

	for key, val := range props {
		field, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		fieldType, _ := field["type"].(string)

		switch fieldType {
		case "string":
			result[key] = generateFakeValue(key)
		case "integer", "number":
			result[key] = gofakeit.Number(1, 1000)
		case "boolean":
			result[key] = gofakeit.Bool()
		case "array":
			items, ok := field["items"].(map[string]interface{})
			if !ok {
				continue
			}

			itemType, _ := items["type"].(string)
			switch itemType {
			case "string":
				result[key] = []string{
					generateFakeValue(key).(string),
					generateFakeValue(key).(string),
				}
			case "number", "integer":
				result[key] = []int{
					gofakeit.Number(1, 1000),
					gofakeit.Number(1, 1000),
				}
			case "boolean":
				result[key] = []bool{
					gofakeit.Bool(),
					gofakeit.Bool(),
				}
			default:
				if ref, ok := items["$ref"].(string); ok {
					refName := resolveRef(ref)
					if subDefRaw, ok := defs[refName]; ok {
						subDef, ok := subDefRaw.(map[string]interface{})
						if ok {
							result[key] = []interface{}{
								generateFakeObject(subDef, defs),
								generateFakeObject(subDef, defs),
							}
						}
					}
				}
			}
		case "object":
			if props, ok := field["properties"].(map[string]interface{}); ok {
				nestedObj := make(map[string]interface{})
				for propKey, propVal := range props {
					propField, ok := propVal.(map[string]interface{})
					if !ok {
						continue
					}

					if propField["type"] == "string" {
						nestedObj[propKey] = generateFakeValue(propKey)
					} else {
						nestedObj[propKey] = gofakeit.Word()
					}
				}
				result[key] = nestedObj
			} else if ref, ok := field["$ref"].(string); ok {
				refName := resolveRef(ref)
				if subDefRaw, ok := defs[refName]; ok {
					subDef, ok := subDefRaw.(map[string]interface{})
					if ok {
						result[key] = generateFakeObject(subDef, defs)
					}
				}
			}
		default:
			if ref, ok := field["$ref"].(string); ok {
				refName := resolveRef(ref)
				if subDefRaw, ok := defs[refName]; ok {
					subDef, ok := subDefRaw.(map[string]interface{})
					if ok {
						result[key] = generateFakeObject(subDef, defs)
					}
				}
			}
		}
	}
	return result
}

// Generate context-aware fake values based on field name
func generateFakeValue(fieldName string) interface{} {
	field := strings.ToLower(fieldName)

	switch {
	case strings.Contains(field, "date"):
		return gofakeit.Date().Format("2006-01-02")
	case strings.Contains(field, "time"):
		return gofakeit.Date().Format("15:04:05")
	case strings.Contains(field, "uuid") || strings.Contains(field, "id"):
		return gofakeit.UUID()
	case strings.Contains(field, "email"):
		return gofakeit.Email()
	case strings.Contains(field, "name") && strings.Contains(field, "first"):
		return gofakeit.FirstName()
	case strings.Contains(field, "name") && strings.Contains(field, "last"):
		return gofakeit.LastName()
	case strings.Contains(field, "name") && !strings.Contains(field, "first") && !strings.Contains(field, "last"):
		return gofakeit.Name()
	case strings.Contains(field, "city"):
		return gofakeit.City()
	case strings.Contains(field, "country"):
		return gofakeit.Country()
	case strings.Contains(field, "phone"):
		return gofakeit.Phone()
	case strings.Contains(field, "postal") || strings.Contains(field, "zip"):
		return gofakeit.Zip()
	case strings.Contains(field, "address"):
		return gofakeit.Address().Address
	case strings.Contains(field, "status"):
		return gofakeit.RandomString([]string{"active", "inactive", "pending"})
	case strings.Contains(field, "description"):
		return gofakeit.Sentence(5)
	case strings.Contains(field, "title"):
		return gofakeit.Sentence(3)
	case strings.Contains(field, "url") || strings.Contains(field, "link"):
		return gofakeit.URL()
	default:
		return gofakeit.Word()
	}
}
