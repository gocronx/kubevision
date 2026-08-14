package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/operation"
	"github.com/gocronx/kubevision/internal/packages"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

type PackageHandler struct {
	service    *packages.Service
	catalog    *packages.Catalog
	upgrades   *packages.UpgradeManager
	operations *operation.Manager
	resolve    func(context.Context, string) (string, error)
}

func (h *PackageHandler) WithOperations(manager *operation.Manager) *PackageHandler {
	h.operations = manager
	return h
}

func (h *PackageHandler) WithUpgradeManager(manager *packages.UpgradeManager) *PackageHandler {
	h.upgrades = manager
	return h
}

func (h *PackageHandler) WithCatalog(catalog *packages.Catalog) *PackageHandler {
	h.catalog = catalog
	return h
}

type packageChangeRequest struct {
	ReleaseName       string                 `json:"releaseName" binding:"required"`
	Namespace         string                 `json:"namespace" binding:"required"`
	Source            packages.ChartSource   `json:"source" binding:"required"`
	Values            map[string]interface{} `json:"values"`
	CreateNamespace   bool                   `json:"createNamespace"`
	Wait              bool                   `json:"wait"`
	Atomic            bool                   `json:"atomic"`
	TimeoutSeconds    int                    `json:"timeoutSeconds"`
	ConfirmationToken string                 `json:"confirmationToken"`
}

type packageUpgradeCheckRequest struct {
	Source *packages.ChartSource `json:"source,omitempty"`
}

func (r packageChangeRequest) options() packages.ChangeOptions {
	return packages.ChangeOptions{ReleaseName: r.ReleaseName, Namespace: r.Namespace, Source: r.Source, Values: r.Values, CreateNamespace: r.CreateNamespace, Wait: r.Wait, Atomic: r.Atomic, Timeout: time.Duration(r.TimeoutSeconds) * time.Second, ConfirmationToken: r.ConfirmationToken}
}

func NewPackageHandler(service *packages.Service, resolve func(context.Context, string) (string, error)) *PackageHandler {
	return &PackageHandler{service: service, resolve: resolve}
}

func (h *PackageHandler) List(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	limit, err := queryInt(c, "limit", 0)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid limit")
		return
	}
	items, err := h.service.List(c.Request.Context(), packageActor(c), cluster, packages.ListOptions{Namespace: c.Query("namespace"), State: c.Query("state"), Label: c.Query("label"), Limit: limit})
	writePackageResult(c, items, err)
}

func (h *PackageHandler) Get(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	revision, err := queryInt(c, "revision", 0)
	if err != nil || revision < 0 {
		response.Error(c, bizerr.CodeParamInvalid, "invalid revision")
		return
	}
	item, err := h.service.Get(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"), revision)
	writePackageResult(c, item, err)
}

func (h *PackageHandler) History(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	items, err := h.service.History(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"))
	writePackageResult(c, items, err)
}

func (h *PackageHandler) Preview(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid package change request")
		return
	}
	item, err := h.service.Preview(c.Request.Context(), packageActor(c), c.Param("operation"), cluster, req.options())
	writePackageResult(c, item, err)
}

func (h *PackageHandler) Install(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid install request")
		return
	}
	h.queueChange(c, "install", cluster, req)
}

func (h *PackageHandler) queueChange(c *gin.Context, action, cluster string, req packageChangeRequest) {
	if h.operations == nil {
		response.Error(c, bizerr.CodeInternal, "operation manager is unavailable")
		return
	}
	actor := packageActor(c)
	prepared, err := h.service.PrepareChange(c.Request.Context(), actor, action, cluster, req.options())
	if err != nil {
		writePackageResult(c, nil, err)
		return
	}
	item, err := h.operations.Submit(c.Request.Context(), packageOperationInput(c, action, cluster, req.Namespace, req.ReleaseName, packages.NewChangeOperationTask(prepared)))
	writePackageResult(c, item, err)
}

func (h *PackageHandler) Upgrade(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid upgrade request")
		return
	}
	h.queueChange(c, "upgrade", cluster, req)
}

func (h *PackageHandler) CheckUpgrade(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageUpgradeCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid upgrade check request")
		return
	}
	item, err := h.service.CheckUpgrade(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"), req.Source)
	writePackageResult(c, item, err)
}

func (h *PackageHandler) ListRepositories(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	items, err := h.catalog.ListRepositories(c.Request.Context(), packageActor(c))
	writePackageResult(c, items, err)
}
func (h *PackageHandler) CreateRepository(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	var req packages.RepositoryInput
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository")
		return
	}
	item, err := h.catalog.SaveRepository(c.Request.Context(), packageActor(c), 0, req)
	writePackageResult(c, item, err)
}
func (h *PackageHandler) UpdateRepository(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("repositoryId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository ID")
		return
	}
	var req packages.RepositoryInput
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository")
		return
	}
	item, saveErr := h.catalog.SaveRepository(c.Request.Context(), packageActor(c), uint(id), req)
	writePackageResult(c, item, saveErr)
}
func (h *PackageHandler) DeleteRepository(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("repositoryId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository ID")
		return
	}
	writePackageResult(c, nil, h.catalog.DeleteRepository(c.Request.Context(), packageActor(c), uint(id)))
}
func (h *PackageHandler) TestRepository(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("repositoryId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository ID")
		return
	}
	writePackageResult(c, nil, h.catalog.TestRepository(c.Request.Context(), packageActor(c), uint(id)))
}
func (h *PackageHandler) RepositoryCharts(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("repositoryId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid repository ID")
		return
	}
	items, listErr := h.catalog.RepositoryCharts(c.Request.Context(), packageActor(c), uint(id), c.Query("q"))
	writePackageResult(c, items, listErr)
}
func (h *PackageHandler) ArtifactSearch(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	limit, _ := queryInt(c, "limit", 20)
	items, err := h.catalog.SearchArtifactHub(c.Request.Context(), packageActor(c), c.Query("q"), limit)
	writePackageResult(c, items, err)
}
func (h *PackageHandler) InspectChart(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	var source packages.ChartSource
	if c.ShouldBindJSON(&source) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid chart source")
		return
	}
	item, err := h.catalog.Inspect(c.Request.Context(), packageActor(c), source)
	writePackageResult(c, item, err)
}
func (h *PackageHandler) UploadChart(c *gin.Context) {
	if h.catalog == nil {
		response.Error(c, bizerr.CodeInternal, "Helm catalog is unavailable")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 51<<20)
	file, header, err := c.Request.FormFile("chart")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "chart file is required")
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".tgz") {
		response.Error(c, bizerr.CodeParamInvalid, "only .tgz Helm charts are accepted")
		return
	}
	item, uploadErr := h.catalog.Upload(c.Request.Context(), packageActor(c), file, header.Size)
	writePackageResult(c, item, uploadErr)
}

func (h *PackageHandler) ListUpgradePolicies(c *gin.Context) {
	if h.upgrades == nil {
		response.Error(c, bizerr.CodeInternal, "upgrade manager is unavailable")
		return
	}
	items, err := h.upgrades.List(c.Request.Context(), packageActor(c))
	writePackageResult(c, items, err)
}
func (h *PackageHandler) CreateUpgradePolicy(c *gin.Context) {
	if h.upgrades == nil {
		response.Error(c, bizerr.CodeInternal, "upgrade manager is unavailable")
		return
	}
	var req packages.UpgradePolicyInput
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid upgrade policy")
		return
	}
	item, err := h.upgrades.Save(c.Request.Context(), packageActor(c), 0, req)
	writePackageResult(c, item, err)
}
func (h *PackageHandler) UpdateUpgradePolicy(c *gin.Context) {
	if h.upgrades == nil {
		response.Error(c, bizerr.CodeInternal, "upgrade manager is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("policyId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid policy ID")
		return
	}
	var req packages.UpgradePolicyInput
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid upgrade policy")
		return
	}
	item, saveErr := h.upgrades.Save(c.Request.Context(), packageActor(c), uint(id), req)
	writePackageResult(c, item, saveErr)
}
func (h *PackageHandler) DeleteUpgradePolicy(c *gin.Context) {
	if h.upgrades == nil {
		response.Error(c, bizerr.CodeInternal, "upgrade manager is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("policyId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid policy ID")
		return
	}
	writePackageResult(c, nil, h.upgrades.Delete(c.Request.Context(), packageActor(c), uint(id)))
}
func (h *PackageHandler) CheckUpgradePolicy(c *gin.Context) {
	if h.upgrades == nil {
		response.Error(c, bizerr.CodeInternal, "upgrade manager is unavailable")
		return
	}
	id, err := strconv.ParseUint(c.Param("policyId"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid policy ID")
		return
	}
	item, checkErr := h.upgrades.CheckNow(c.Request.Context(), packageActor(c), uint(id))
	writePackageResult(c, item, checkErr)
}

type packageRollbackRequest struct {
	Revision       int  `json:"revision" binding:"required,min=1"`
	Wait           bool `json:"wait"`
	Atomic         bool `json:"atomic"`
	TimeoutSeconds int  `json:"timeoutSeconds"`
}

func (h *PackageHandler) Rollback(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageRollbackRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "revision is required")
		return
	}
	task := packages.OperationTask{Action: "rollback", Cluster: cluster, Namespace: c.Param("namespace"), Name: c.Param("name"), Revision: req.Revision, Wait: req.Wait, Atomic: req.Atomic, TimeoutSeconds: req.TimeoutSeconds}
	h.queueTask(c, task)
}

type packageRemoveRequest struct {
	Confirmation   string `json:"confirmation" binding:"required"`
	KeepHistory    bool   `json:"keepHistory"`
	Wait           bool   `json:"wait"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func (h *PackageHandler) Remove(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageRemoveRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "confirmation is required")
		return
	}
	task := packages.OperationTask{Action: "remove", Cluster: cluster, Namespace: c.Param("namespace"), Name: c.Param("name"), Confirmation: req.Confirmation, KeepHistory: req.KeepHistory, Wait: req.Wait, TimeoutSeconds: req.TimeoutSeconds}
	h.queueTask(c, task)
}

func (h *PackageHandler) queueTask(c *gin.Context, task packages.OperationTask) {
	if h.operations == nil {
		response.Error(c, bizerr.CodeInternal, "operation manager is unavailable")
		return
	}
	item, err := h.operations.Submit(c.Request.Context(), packageOperationInput(c, task.Action, task.Cluster, task.Namespace, task.Name, task))
	writePackageResult(c, item, err)
}

func packageOperationInput(c *gin.Context, action, cluster, namespace, name string, payload interface{}) operation.Input {
	return operation.Input{UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c), Kind: packages.OperationKind,
		Action: action, Cluster: cluster, Namespace: namespace, ResourceName: name, RequestID: c.GetString("requestID"), Payload: payload}
}

func (h *PackageHandler) cluster(c *gin.Context) (string, bool) {
	cluster, err := h.resolve(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, bizerr.CodeNotFound, "cluster not found")
		return "", false
	}
	return cluster, true
}

func packageActor(c *gin.Context) packages.Actor {
	return packages.Actor{UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c), Role: middleware.GetUserRole(c), ClientIP: c.ClientIP()}
}
func queryInt(c *gin.Context, name string, fallback int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}
func writePackageResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		response.Success(c, data)
		return
	}
	if b, ok := err.(*bizerr.BizError); ok {
		response.ErrorWithBizErr(c, b)
		return
	}
	response.Error(c, bizerr.CodeInternal, "internal server error")
}
