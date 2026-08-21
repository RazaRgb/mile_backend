package utils

import (
	"backend/src/models"
	"fmt"
	"strings"
)

type prompt struct {
	System string
	User   string
}

// rootTopic returns the topmost topic of a learning path — the first item of
// the "root → … → node" chain built by contextChain. Used to anchor every
// prompt to the stream's overall subject so content can't drift off-topic.
func rootTopic(learningPath string) string {
	if idx := strings.Index(learningPath, " → "); idx >= 0 {
		return learningPath[:idx]
	}
	return learningPath
}

func CreateExpansionPrompt(node models.Node, learningPath string) prompt {
	return prompt{
		System: fmt.Sprintf(
			"The overall topic of this learning path is %q (the first item in the path). Split the given topic into subtopics that serve this overall topic directly. Never introduce subtopics about tangential tools, methods, or fields that the overall topic does not require. Keep the subtopics a single level below the topic and do not skip any level.",
			rootTopic(learningPath),
		),
		User: fmt.Sprintf("The topic to subdivide is: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
	}
}

func CreateGenerationPrompt(node models.Node, learningPath string) prompt {
	return prompt{
		System: fmt.Sprintf(
			"You are an expert technical writer. Write a concise, self-contained micro-lesson (under 2 minutes of reading) about the given topic. The overall topic of this learning path is %q (the first item in the path); the lesson must serve that overall topic — do not drift into tangential material the overall topic does not require. Use short paragraphs and keep it engaging and easy to read through.",
			rootTopic(learningPath),
		),
		User: fmt.Sprintf("The topic for the article is: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
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
		System: fmt.Sprintf(
			"You are a content planner for a micro-learning app. The overall topic of this learning path is %q (the first item in the path). Judge whether the given topic is narrow enough to be explained fully and completely in under 100 words, in the context of that overall topic. Be strict: if properly covering the topic would require more than 100 words, the answer must be false.",
			rootTopic(learningPath),
		),
		User: fmt.Sprintf("Topic: %s.\nLearning path (parent topics first): %s", node.Topic, learningPath),
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
