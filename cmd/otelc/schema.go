package main

const metricConfigSchema = `
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "metrics": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string"
          },
          "type": {
            "type": "string",
            "enum": ["counter", "gauge", "histogram", "summary"]
          },
          "description": {
            "type": "string"
          },
          "help": {
            "type": "string"
          },
          "labels": {
            "type": "array",
            "items": {
              "type": "string"
            }
          },
          "buckets": {
            "type": "array",
            "items": {
              "type": "number"
            }
          }
        },
        "required": ["name", "type"],
        "allOf": [
          {
            "if": {
              "properties": {
                "type": {
                  "const": "histogram"
                }
              }
            },
            "then": {
              "properties": {
                "buckets": {
                  "type": "array",
                  "items": {
                    "type": "number"
                  }
                }
              }
            },
            "else": {
              "properties": {
                "buckets": {
                  "type": "null"
                }
              }
            }
          }
        ],
        "additionalProperties": false
      }
    }
  },
  "required": ["metrics"]
}
`
