package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

type Feature struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Details     string `json:"details"`
	Upvotes     int    `json:"upvotes"`
	Completed   bool   `json:"completed"`
}

type FeatureSummary struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Upvotes   int    `json:"upvotes"`
	Completed bool   `json:"completed"`
}

type FeatureRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Details     string `json:"details"`
}

type VoteRequest struct {
	FeatureID int `json:"feature_id"`
}

type CompleteRequest struct {
	FeatureID int `json:"feature_id"`
}

var (
	features = map[int]*Feature{
		1: {ID: 1, Title: "Dark Mode", Description: "Add dark theme support to the application", Details: "Implement a comprehensive dark mode that includes:\n\n- Automatic detection of system preference\n- Manual toggle in user settings\n- Dark variants for all UI components including buttons, forms, modals, and navigation\n- Proper contrast ratios for accessibility compliance\n- Smooth transitions between light and dark modes\n- Persistence of user preference across sessions\n- Support for custom accent colors in dark mode\n\nThis feature should integrate seamlessly with the existing design system and maintain consistency across all pages and components.", Upvotes: 142, Completed: false},
		2: {ID: 2, Title: "Mobile App", Description: "Native mobile application for iOS and Android", Details: "Develop native mobile applications for both iOS and Android platforms:\n\n**iOS App:**\n- Swift/SwiftUI implementation\n- iOS 14+ compatibility\n- App Store submission and compliance\n- Push notifications support\n- Offline functionality for core features\n\n**Android App:**\n- Kotlin implementation\n- Material Design 3 compliance\n- Android 8+ compatibility\n- Google Play Store submission\n- Background sync capabilities\n\n**Shared Features:**\n- Biometric authentication (Face ID, Touch ID, Fingerprint)\n- Deep linking support\n- Synchronized data across web and mobile\n- Performance optimization for battery life\n- Comprehensive testing on multiple devices", Upvotes: 98, Completed: false},
		3: {ID: 3, Title: "API Integration", Description: "Third-party API integrations for popular services", Details: "Build robust integrations with popular third-party services:\n\n**Communication APIs:**\n- Slack workspace integration\n- Microsoft Teams connector\n- Discord webhook support\n- Email service providers (SendGrid, Mailgun)\n\n**Productivity Tools:**\n- Google Workspace (Docs, Sheets, Calendar)\n- Microsoft Office 365\n- Trello and Asana project management\n- Notion database sync\n\n**Development Tools:**\n- GitHub repository integration\n- GitLab CI/CD webhooks\n- Jira issue tracking\n- Jenkins build notifications\n\n**Technical Requirements:**\n- OAuth 2.0 authentication flows\n- Rate limiting and retry mechanisms\n- Webhook validation and security\n- API key management interface\n- Real-time status monitoring\n- Comprehensive error handling and logging", Upvotes: 76, Completed: false},
		4: {ID: 4, Title: "Real-time Chat", Description: "Built-in real-time messaging system", Details: "Implement a comprehensive real-time messaging system:\n\n**Core Features:**\n- Instant messaging with WebSocket connections\n- Group chat rooms and private messaging\n- File sharing (images, documents, code snippets)\n- Message history and search functionality\n- Typing indicators and read receipts\n- Emoji reactions and custom emojis\n\n**Advanced Features:**\n- Message threading for organized discussions\n- Voice and video calling integration\n- Screen sharing capabilities\n- Message encryption for security\n- Customizable notifications\n- Message formatting (markdown support)\n\n**Technical Implementation:**\n- Scalable WebSocket infrastructure\n- Message persistence and backup\n- Real-time presence indicators\n- Mobile push notifications\n- Moderation tools and user management\n- Integration with existing user authentication", Upvotes: 54, Completed: false},
		5: {ID: 5, Title: "Advanced Analytics", Description: "Detailed analytics dashboard with custom metrics", Details: "Create a powerful analytics platform with comprehensive insights:\n\n**Dashboard Features:**\n- Customizable widget layout\n- Real-time data visualization\n- Interactive charts and graphs\n- Drill-down capabilities for detailed analysis\n- Export functionality (PDF, Excel, CSV)\n- Scheduled report generation\n\n**Metrics and KPIs:**\n- User engagement tracking\n- Performance monitoring\n- Conversion funnel analysis\n- A/B testing results\n- Custom event tracking\n- Revenue and growth metrics\n\n**Advanced Capabilities:**\n- Machine learning insights and predictions\n- Anomaly detection and alerts\n- Cohort analysis and user segmentation\n- Custom query builder\n- API for programmatic access\n- Integration with Google Analytics and other tools\n\n**Technical Features:**\n- High-performance data processing\n- Real-time data streaming\n- Historical data retention policies\n- GDPR compliance and data privacy controls", Upvotes: 31, Completed: false},
	}
	nextID = 6
	mu     sync.RWMutex
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /features/top", getTopFeature)
	mux.HandleFunc("GET /features/{id}", getFeatureDetails)
	mux.HandleFunc("POST /features", addFeature)
	mux.HandleFunc("POST /features/vote", voteForFeature)
	mux.HandleFunc("POST /features/complete", completeFeature)
	mux.HandleFunc("DELETE /features/{id}", deleteFeature)
	mux.HandleFunc("GET /features", getAllFeatures)
	mux.HandleFunc("GET /openapi.json", getOpenAPISpec)

	fmt.Println("Feature request server starting on :9090")
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /features/top      - Get most voted feature (summary)")
	fmt.Println("  GET    /features/{id}     - Get feature details")
	fmt.Println("  POST   /features          - Add new feature")
	fmt.Println("  POST   /features/vote     - Vote for a feature")
	fmt.Println("  POST   /features/complete - Mark feature as completed")
	fmt.Println("  DELETE /features/{id}     - Delete a feature")
	fmt.Println("  GET    /features          - Get all features (summaries)")
	fmt.Println("  GET    /openapi.json      - Get OpenAPI specification")

	err := http.ListenAndServe(":9090", mux)
	fmt.Printf("error: %s\n", err.Error())
}

func getTopFeature(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	var topFeature *Feature
	maxVotes := -1

	for _, feature := range features {
		if feature.Upvotes > maxVotes {
			maxVotes = feature.Upvotes
			topFeature = feature
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if topFeature == nil {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "No features found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	summary := FeatureSummary{
		ID:        topFeature.ID,
		Title:     topFeature.Title,
		Upvotes:   topFeature.Upvotes,
		Completed: topFeature.Completed,
	}
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func addFeature(w http.ResponseWriter, r *http.Request) {
	var req FeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	if req.Title == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	mu.Lock()
	feature := &Feature{
		ID:          nextID,
		Title:       req.Title,
		Description: req.Description,
		Details:     req.Details,
		Upvotes:     0,
		Completed:   false,
	}
	features[nextID] = feature
	nextID++
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(feature); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func voteForFeature(w http.ResponseWriter, r *http.Request) {
	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	mu.Lock()
	feature, exists := features[req.FeatureID]
	if !exists {
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Feature not found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	feature.Upvotes++
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feature); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func getFeatureDetails(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid feature ID"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	mu.RLock()
	feature, exists := features[id]
	mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Feature not found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	if err := json.NewEncoder(w).Encode(feature); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func getAllFeatures(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	featureList := make([]*Feature, 0, len(features))
	for _, feature := range features {
		featureList = append(featureList, feature)
	}

	sort.Slice(featureList, func(i, j int) bool {
		return featureList[i].Upvotes > featureList[j].Upvotes
	})

	summaries := make([]FeatureSummary, len(featureList))
	for i, feature := range featureList {
		summaries[i] = FeatureSummary{
			ID:        feature.ID,
			Title:     feature.Title,
			Upvotes:   feature.Upvotes,
			Completed: feature.Completed,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func getOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	// Get the path to the OpenAPI spec file
	// When deployed with ko, KO_DATA_PATH points to the kodata directory
	// When running locally, fall back to the local kodata directory
	var specPath string
	if koDataPath := os.Getenv("KO_DATA_PATH"); koDataPath != "" {
		specPath = filepath.Join(koDataPath, "openapi.json")
	} else {
		specPath = filepath.Join("kodata", "openapi.json")
	}

	file, err := os.Open(specPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "OpenAPI spec not found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			http.Error(w, "Failed to close file", http.StatusInternalServerError)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Failed to copy file content", http.StatusInternalServerError)
		return
	}
}

func deleteFeature(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid feature ID"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	mu.Lock()
	_, exists := features[id]
	if !exists {
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Feature not found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	delete(features, id)
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

func completeFeature(w http.ResponseWriter, r *http.Request) {
	var req CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	mu.Lock()
	feature, exists := features[req.FeatureID]
	if !exists {
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Feature not found"}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	feature.Completed = true
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feature); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
