package utils

import (
	"backend/src/models"
	"fmt"
)

type prompt struct {
	System string
	User   string
}

func CreateExpansionPrompt(node models.Node, learningPath string) prompt {
	return prompt{
		System: "You will be given a topic/ subtopic, you are required to split it into subtopics, make sure the subtopics are just a single level below the original topic, do not skip any level. Keep the surrounding learning context in mind so the subtopics fit naturally within it.",
		User:   fmt.Sprintf("The topic to subdivide is: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
	}
}

func CreateGenerationPrompt(node models.Node, learningPath string) prompt {
	return prompt{
		System: "You are an expert technical writer. Write a concise, self-contained micro-lesson (under 2 minutes of reading) about the given topic, keeping the surrounding learning context in mind. Use short paragraphs and keep it engaging and easy to read through.",
		User:   fmt.Sprintf("The topic for the article is: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
	}
}

var ExpandPrompt = prompt{
	System: "You will be given a chain of topics, each topic being a subtopic of the previous one. You are required to splthe last topic further into multiple subtopics, make sure the generated subtopics are relevant to the given chain of topics",
	User:   "",
}

var ExpansionSchema string = `
{
  "type": "object",
  "properties": {
    "subtopics": {
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  },
  "required": ["subtopics"]
}
`

func CreateLeafVerificationPrompt(node models.Node, learningPath string) prompt {
	return prompt{
		System: "You are a content planner for a micro-learning app. Judge whether the given topic is narrow enough to be explained fully and completely in under 100 words, given its position in the learning path. Be strict: if properly covering the topic would require more than 100 words, the answer must be false.",
		User:   fmt.Sprintf("Topic: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
	}
}

var LeafVerificationSchema string = `
{
  "type": "object",
  "properties": {
    "can_explain_under_100_words": {
      "type": "boolean"
    }
  },
  "required": ["can_explain_under_100_words"]
}
`
