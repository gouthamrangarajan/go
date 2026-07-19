package services

import (
	"context"
	"datastar-web-learnings/models"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var PROMPT_TO_CHECK_TECH_RELATED_SEARCH = `You will be given a search query from a user. Your task is to decide whether this query is related to technology or tech topics.

Consider these as related to technology:
- General tech concepts (e.g., programming, AI, blockchain, gadgets)
- Technology companies or products
- Names of technology personalities, developers, engineers, or influencers (e.g., "Evan You", "Dan Abramov")
- Technology events or conferences
- Anything involving software, hardware, coding, IT, or digital innovations

If the query is related to technology, respond with: YES  
If it is not related to technology, respond with: NO  

Do not provide any explanation or additional text, only YES or NO.

Example:  
Query: "Evan You"  
Answer: YES

Query: "Best cooking recipes"  
Answer: NO

Now, decide for this query:  
"%v"`

var PROMPT2_TO_CHECK_TECH_RELATED_SEARCH = `You are an intelligent search query optimizer for a technology video library.
Your primary task is to assess a user's search query.

**Part 1: Technology Relevance Check**
First, determine if the query is primarily related to "technology" in a broad sense. This includes programming, software development, hardware, gadgets, AI, machine learning, data science, cybersecurity, cloud computing, specific tech products (like Python, JavaScript, AWS, iPhone, Nvidia GPUs), tech companies, **and the names of prominent figures or creators within the technology domain (e.g., Linus Torvalds, Elon Musk, Grace Hopper, Evan You)**.

**Part 2: Query Optimization (if technology-related)**
If the query is technology-related, suggest an optimized version of the query for vector similarity search. The optimized query should be:
- More descriptive and comprehensive.
- Include relevant keywords or concepts that clarify the user's intent.
- Avoid conversational filler words.
- Expand abbreviations if commonly understood (e.g., "AI" -> "Artificial Intelligence").
- **Crucially, if the original query is a person's name (e.g., "Evan You", "Linus Torvalds"), expand it to something like "videos by Evan You" or "contributions of Linus Torvalds" to better capture intent for video search.**

If the query is NOT technology-related, the optimized query should be an empty string.

**Output Format:**
Respond with two lines.
Line 1: "TECHNOLOGY" or "NOT_TECHNOLOGY"
Line 2: The optimized search query (or an empty string if NOT_TECHNOLOGY)

Examples:

Query: "how to build a website"
TECHNOLOGY
web development tutorial website creation from scratch

Query: "best coffee recipes"
NOT_TECHNOLOGY


Query: "machine learning explained"
TECHNOLOGY
explain machine learning concepts and applications

Query: "by Elon Musk"
TECHNOLOGY
videos by Elon Musk interviews presentations

Query: "latest iPhone release"
TECHNOLOGY
latest Apple iPhone model review features

Query: "history of ancient Rome"
NOT_TECHNOLOGY


Query: "Vue.js tutorial"
TECHNOLOGY
Vue JavaScript framework tutorial guide

Query: "Evan You"
TECHNOLOGY
videos by Evan You Vue.js creator

Query: "what is blockchain"
TECHNOLOGY
explain blockchain technology concepts decentralized ledger

Query: "how to tie a knot"
NOT_TECHNOLOGY


Query: "Nvidia GPU review"
TECHNOLOGY
Nvidia graphics card GPU review performance

Query: "healthy breakfast ideas"
NOT_TECHNOLOGY


Query: "Linus Torvalds contributions"
TECHNOLOGY
Linus Torvalds open source Linux contributions

Query: "cloud security best practices"
TECHNOLOGY
cloud computing security best practices implementation guide

Query: "%v"`

const PROMPT_TO_GENERATE_QUIZ = `
You are a technical communication coach.

Create a quiz using only the supplied video transcript. The learner's goal is
to explain technical concepts clearly in spoken English.

Requirements:
1. Do not introduce facts that are not supported by the transcript.
2. Generate 5 to 8 useful questions.
3. Focus on definitions, purpose, comparisons, processes, and examples.
4. For every question, provide:
   - a concise answer
   - a natural spoken answer
   - important key terms
   - an exact supporting excerpt
   - a timestamp when available
5. Ignore promotions, sponsorships, calls to action, and unrelated introductions.
6. Do not create a question if the transcript does not contain a clear answer.
7. Keep spoken answers between 2 and 4 sentences.
8. Return only JSON matching the supplied schema.

VIDEO TITLE:
%v

TRANSCRIPT:
%v

OUTPUT FORMAT:
{
  "summary": string,
  "talkingPoints": [
    {
      "title": string,
      "text": string
    }
  ],
  "questions": [
    {
      "id": string e.g "q1",
      "type": string,
      "difficulty": string e.g "beginner", "intermediate", "advanced",
      "question": string,
      "shortAnswer": string,
      "speakingAnswer": string,
      "keyTerms": [string],
      "sourceExcerpt": string,
      "startSeconds": string
    }
  ]
}

`

const QUIZ_MOCK_DATA = `{"summary": "This quiz covers five essential technical terms related to Agentic AI: agents.md, agent skills, the Model Context Protocol (MCP), Agent-to-Agent (A2A) communication, and sub-agents. It focuses on how these components enable AI agents to plan tasks, use tools, and collaborate autonomously.",
  "talkingPoints": [
    {
      "title": "Defining an AI Agent",
      "text": "An AI agent is created when an instruction layer is wrapped around a large language model, moving it beyond simple conversation to active task execution."
    },
    {
      "title": "Standards and Protocols",
      "text": "The industry is moving toward open standards like MCP and A2A to ensure different agents and tools can communicate seamlessly through shared interfaces."
    }
  ],
  "questions": [
    {
      "id": "q1",
      "type": "definition",
      "difficulty": "beginner",
      "question": "What is agents.md and what is its primary purpose?",
      "shortAnswer": "A markdown text file at a project's root that provides specific instructions and conventions for an AI agent.",
      "speakingAnswer": "Think of agents.md as a readme file specifically written for AI agents rather than humans. It sits at the root of a project and tells the agent which commands to run or which coding conventions to follow. The agent reads this file to understand the specific context of the codebase it is working on.",
      "keyTerms": [
        "markdown",
        "root",
        "instruction layer",
        "coding conventions"
      ],
      "sourceExcerpt": "It's just a text file. It sits at the root of a project and whenever the agent starts work in that project, it reads whatever is in that agents.md file. Now the file tells the agent things like which commands to run for tests or which coding conventions this codebase uses.",
      "startSeconds": "0:56"
    },
    {
      "id": "q2",
      "type": "process",
      "difficulty": "intermediate",
      "question": "How does an AI agent know when to utilize an 'agent skill'?",
      "shortAnswer": "An agent skill contains a skill.md file with metadata that describes specifically when the agent should invoke that skill.",
      "speakingAnswer": "An agent skill is contained within a folder that includes a file called skill.md. This file contains metadata and a description that essentially tells the agent, 'invoke me when the user wants to do x.' It allows the agent to call upon specific knowledge or capabilities only when they are contextually relevant.",
      "keyTerms": [
        "agent skill",
        "skill.md",
        "metadata",
        "invoke"
      ],
      "sourceExcerpt": "Inside skill.md is some metadata including a description and that tells the agent something like invoke me when the user wants to x.",
      "startSeconds": "3:44"
    },
    {
      "id": "q3",
      "type": "definition",
      "difficulty": "intermediate",
      "question": "What is the Model Context Protocol (MCP) and why is it useful?",
      "shortAnswer": "An open protocol for connecting AI applications to tools and data sources through a standard interface.",
      "speakingAnswer": "MCP, or Model Context Protocol, is an open standard used to connect AI agents to external tools, data sources, and workflows. It uses an MCP server to wrap these tools into a standard interface. This allows any agent that speaks the protocol to communicate with those tools without needing custom integrations.",
      "keyTerms": [
        "MCP",
        "open protocol",
        "standard interface",
        "MCP server"
      ],
      "sourceExcerpt": "MCP is an open protocol for connecting AI applications to tools and data sources and workflows and it comes with something called an MCP server. An MCP server wraps up a tool or a data source into a standard interface.",
      "startSeconds": "5:16"
    },
    {
      "id": "q4",
      "type": "process",
      "difficulty": "advanced",
      "question": "In the A2A (Agent-to-Agent) protocol, how do agents identify each other's capabilities?",
      "shortAnswer": "Agents publish an 'agent card' which describes what the agent does and how to communicate with it.",
      "speakingAnswer": "In the Agent-to-Agent protocol, agents communicate by publishing what is known as an agent card. This card serves as a description of the agent's specific functions and the technical instructions on how to talk to it. This standard allows different agents to collaborate autonomously.",
      "keyTerms": [
        "A2A",
        "agent card",
        "communication",
        "open protocol"
      ],
      "sourceExcerpt": "With A to A, each agent publishes something called an agent card. And that's just basically a description of what the agent does and how to talk to it.",
      "startSeconds": "7:14"
    },
    {
      "id": "q5",
      "type": "comparison",
      "difficulty": "intermediate",
      "question": "What defines a sub-agent and how does it differ from other agent standards?",
      "shortAnswer": "A child agent spawned for a specific task that runs in its own context window, though it lacks a formal standard document.",
      "speakingAnswer": "A sub-agent is a child agent spawned by a main agent to handle a specific piece of work in its own fresh context window. Unlike MCP or A2A, sub-agents are considered a common architectural pattern rather than a formal industry standard. They are primarily used when one agent isn't enough to handle a complex task.",
      "keyTerms": [
        "sub-agent",
        "child agent",
        "context window",
        "common pattern"
      ],
      "sourceExcerpt": "A sub-agent is a child agent that the main agent spawns to do a specific piece of work and each sub-agent runs in its own fresh context window. Now, sub-agents are a little bit different from the other four terms because sub-agents are a common pattern in modern agent systems, but they don't really have a formal standard document behind them.",
      "startSeconds": "9:00"
    }
  ]
}`

const VECTOR_DATA_TEMPLATE_WITH_TRANSCRIPT = `Title: %v
Subtitle: %v
Tags: %v
Description: %v
Transcript: %v`

const VECTOR_DATA_TEMPLATE = `Title: %v
Subtitle: %v
Tags: %v
Description: %v`

func ConstructTextToVectorize(data models.VideoResponse, description string) models.VideoResponse {
	if strings.TrimSpace(data.Transcript) != "" {
		data.TextToVectorize = fmt.Sprintf(VECTOR_DATA_TEMPLATE_WITH_TRANSCRIPT, data.Title, data.Subtitle, strings.Join(data.Tags, ", "), description, data.Transcript)
	} else {
		data.TextToVectorize = fmt.Sprintf(VECTOR_DATA_TEMPLATE, data.Title, data.Subtitle, strings.Join(data.Tags, ", "), description)
	}
	data.DescriptionFromYTAPI = description
	return data
}
func GetFirstSetOfVideos(ctxt context.Context) []models.VideoResponse {
	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	channel := make(chan []models.VideoResponse)
	go GetVideos(ctxt, models.GetVideosRequest{Limit: noOfItems, Offset: 0}, channel)
	defer close(channel)
	videos := <-channel
	return videos
}

func removeJSONCodeFence(input string) string {
	output := strings.TrimSpace(input)
  output,_ = strings.CutPrefix(output, "```json")
  output,_ = strings.CutPrefix(output, "```JSON")
  output,_ = strings.CutPrefix(output, "```")
	output = strings.TrimSpace(output)
	output,_ = strings.CutSuffix(output, "```")
	return strings.TrimSpace(output)
}
