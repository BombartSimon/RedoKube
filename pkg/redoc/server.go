package redoc

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-openapi/loads"
	"github.com/gorilla/mux"
	"k8s.io/klog/v2"

	docsv1 "github.com/BombartSimon/redokube/api/v1"
	"github.com/BombartSimon/redokube/pkg/mockers"
)

const (
	defaultPort = 8080
	redocHTML   = `<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-url="{{ .SpecURL }}"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>`
)

// Server represents the documentation server that serves OpenAPI specs with Redoc
type Server struct {
	router        *mux.Router
	server        *http.Server
	specs         map[string]*SpecInfo
	specsMutex    sync.RWMutex
	port          int
	externalURL   string
	specDirectory string
}

// SpecInfo holds information about a registered OpenAPI spec
type SpecInfo struct {
	Title    string
	SpecPath string
	SpecURL  string
	Document *loads.Document
}

// NewServer creates a new documentation server
func NewServer(options ...ServerOption) *Server {
	s := &Server{
		router:        mux.NewRouter(),
		specs:         make(map[string]*SpecInfo),
		port:          defaultPort,
		specDirectory: "/tmp/redokube-specs", // Default directory to store specs
	}

	// Apply options
	for _, opt := range options {
		opt(s)
	}

	// Ensure spec directory exists
	if err := os.MkdirAll(s.specDirectory, 0755); err != nil {
		klog.Fatalf("Failed to create spec directory: %v", err)
	}

	// Setup routes
	s.router.PathPrefix("/specs/").Handler(http.StripPrefix("/specs/", http.FileServer(http.Dir(s.specDirectory))))
	s.router.HandleFunc("/docs/{name}", s.handleDoc)
	s.router.HandleFunc("/", s.handleIndex)

	// Setup server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	return s
}

// ServerOption represents a server configuration option
type ServerOption func(*Server)

// WithPort sets the server port
func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

// WithExternalURL sets the external URL for the server
func WithExternalURL(url string) ServerOption {
	return func(s *Server) {
		s.externalURL = url
	}
}

// WithSpecDirectory sets the directory to store spec files
func WithSpecDirectory(dir string) ServerOption {
	return func(s *Server) {
		s.specDirectory = dir
	}
}

// Start starts the documentation server
func (s *Server) Start() error {
	klog.Infof("Starting Redokube documentation server on port %d", s.port)
	return s.server.ListenAndServe()
}

// Stop stops the documentation server
func (s *Server) Stop() error {
	klog.Info("Stopping Redokube documentation server")
	return s.server.Close()
}

// RegisterSpec registers an OpenAPI spec from a CRD
func (s *Server) RegisterSpec(openAPISpec *docsv1.OpenAPISpec) (string, error) {
	s.specsMutex.Lock()
	defer s.specsMutex.Unlock()

	name := fmt.Sprintf("%s-%s", openAPISpec.Namespace, openAPISpec.Name)
	specPath := openAPISpec.Spec.SpecPath
	specContent := openAPISpec.Spec.SpecContent
	isMockEnabled := openAPISpec.Spec.Mock

	// Create spec filename
	specFilename := fmt.Sprintf("%s.json", name)
	specFilePath := filepath.Join(s.specDirectory, specFilename)

	// Check if we have direct content or need to fetch from path
	if specContent != "" {
		klog.Infof("Using direct OpenAPI spec content for %s", name)

		// Apply mocking if enabled
		if isMockEnabled {
			klog.Infof("Mock is enabled for %s, generating fake examples", name)
			mockedContent, err := mockers.MockOpenAPISpec(specContent)
			if err != nil {
				klog.Warningf("Failed to generate mock data: %v. Using original content.", err)
			} else {
				specContent = mockedContent
				klog.Info("Successfully generated mock examples")
			}
		}

		// Create the spec file
		specFile, err := os.Create(specFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create spec file: %v", err)
		}
		defer specFile.Close()

		// Write the content directly to file
		_, err = specFile.WriteString(specContent)
		if err != nil {
			return "", fmt.Errorf("failed to write spec content to file: %v", err)
		}
	} else if specPath != "" {
		// If it's a URL, download directly to maintain the exact format
		if strings.HasPrefix(specPath, "http://") || strings.HasPrefix(specPath, "https://") {
			klog.Infof("Downloading OpenAPI spec from URL: %s", specPath)
			resp, err := http.Get(specPath)
			if err != nil {
				return "", fmt.Errorf("failed to download OpenAPI spec from URL %s: %v", specPath, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("failed to download OpenAPI spec from URL %s: status code %d", specPath, resp.StatusCode)
			}

			// Read the content for potential mocking
			content, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read spec content: %v", err)
			}

			// Apply mocking if enabled
			if isMockEnabled {
				klog.Infof("Mock is enabled for %s, generating fake examples", name)
				mockedContent, err := mockers.MockOpenAPISpec(string(content))
				if err != nil {
					klog.Warningf("Failed to generate mock data: %v. Using original content.", err)
				} else {
					content = []byte(mockedContent)
					klog.Info("Successfully generated mock examples")
				}
			}

			// Create the spec file
			specFile, err := os.Create(specFilePath)
			if err != nil {
				return "", fmt.Errorf("failed to create spec file: %v", err)
			}
			defer specFile.Close()

			// Write the content to file
			_, err = specFile.Write(content)
			if err != nil {
				return "", fmt.Errorf("failed to write spec to file: %v", err)
			}
		} else {
			// For local files, read content for potential mocking
			sourceFile, err := os.Open(specPath)
			if err != nil {
				return "", fmt.Errorf("failed to open OpenAPI spec file %s: %v", specPath, err)
			}

			content, err := io.ReadAll(sourceFile)
			sourceFile.Close() // Close after reading

			if err != nil {
				return "", fmt.Errorf("failed to read spec content: %v", err)
			}

			// Apply mocking if enabled
			if isMockEnabled {
				klog.Infof("Mock is enabled for %s, generating fake examples", name)
				mockedContent, err := mockers.MockOpenAPISpec(string(content))
				if err != nil {
					klog.Warningf("Failed to generate mock data: %v. Using original content.", err)
				} else {
					content = []byte(mockedContent)
					klog.Info("Successfully generated mock examples")
				}
			}

			// Create the spec file
			specFile, err := os.Create(specFilePath)
			if err != nil {
				return "", fmt.Errorf("failed to create spec file: %v", err)
			}
			defer specFile.Close()

			// Write the content to file
			_, err = specFile.Write(content)
			if err != nil {
				return "", fmt.Errorf("failed to write spec to file: %v", err)
			}
		}
	} else {
		return "", fmt.Errorf("neither specPath nor specContent provided in OpenAPISpec %s", name)
	}

	// Try to load the spec to validate it (but we don't modify it)
	document, err := loads.Spec(specFilePath)
	if err != nil {
		// Log warning but continue
		klog.Warningf("OpenAPI spec loaded from %s might not be valid: %v", name, err)
	}

	// Build spec URL
	baseURL := s.externalURL
	if baseURL == "" {
		// If no external URL is set, use the service's cluster DNS name
		baseURL = fmt.Sprintf("http://redokube.%s.svc:%d", openAPISpec.Namespace, s.port)
	}

	specInfo := &SpecInfo{
		Title:    openAPISpec.Spec.Title,
		SpecPath: specPath,
		SpecURL:  fmt.Sprintf("%s/specs/%s", baseURL, specFilename),
		Document: document,
	}

	s.specs[name] = specInfo

	// Return the documentation URL
	return fmt.Sprintf("%s/docs/%s", baseURL, name), nil
}

// handleDoc handles requests for specific API documentation
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	s.specsMutex.RLock()
	specInfo, ok := s.specs[name]
	s.specsMutex.RUnlock()

	if !ok {
		http.Error(w, "API documentation not found", http.StatusNotFound)
		return
	}

	// Render Redoc template
	tmpl, err := template.New("redoc").Parse(redocHTML)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title   string
		SpecURL string
	}{
		Title:   specInfo.Title,
		SpecURL: specInfo.SpecURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// handleIndex handles requests to the root path
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.specsMutex.RLock()
	defer s.specsMutex.RUnlock()

	if len(s.specs) == 0 {
		fmt.Fprintf(w, "<html><body><h1>Redokube Documentation</h1><p>No API documentation available yet.</p></body></html>")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<html><body><h1>Redokube Documentation</h1><ul>")

	for name, specInfo := range s.specs {
		fmt.Fprintf(w, "<li><a href=\"/docs/%s\">%s</a></li>", name, specInfo.Title)
	}

	fmt.Fprintf(w, "</ul></body></html>")
}
