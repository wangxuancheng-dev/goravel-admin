package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	apperrors "goravel/app/errors"
	"goravel/app/models"
)

const generatedModuleManifestDir = "database/seeders/modules"

// ModuleInstallResult summarizes what was installed.
type ModuleInstallResult struct {
	MenuID        uint   `json:"menu_id"`
	MenuSlug      string `json:"menu_slug"`
	PermissionIDs []uint `json:"permission_ids"`
	ManifestPath  string `json:"manifest_path,omitempty"`
}

type ModuleInstaller interface {
	Install(manifest *ModuleManifest) (*ModuleInstallResult, error)
	SyncManifestFile(manifest *ModuleManifest) (string, error)
}

type ModuleInstallerImpl struct {
	ctx context.Context
}

func NewModuleInstaller(ctx context.Context) ModuleInstaller {
	return &ModuleInstallerImpl{ctx: ctx}
}

func (i *ModuleInstallerImpl) Install(manifest *ModuleManifest) (*ModuleInstallResult, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}

	if len(manifest.Permissions) == 0 {
		rebuilt, err := BuildModuleManifest(manifest.ModuleName, manifest.TableName, map[string]bool{
			"has_create": manifest.HasCreate,
			"has_edit":   manifest.HasEdit,
			"has_delete": manifest.HasDelete,
			"has_export": manifest.HasExport,
			"has_import": manifest.HasImport,
		}, &ModuleInstallConfig{
			MenuTitle:      manifest.MenuTitle,
			ParentMenuSlug: manifest.ParentMenuSlug,
			MenuSort:       manifest.MenuSort,
		})
		if err != nil {
			return nil, err
		}
		manifest.Permissions = rebuilt.Permissions
	}

	menu, err := i.upsertMenu(manifest)
	if err != nil {
		return nil, err
	}

	permissionIDs, err := i.upsertPermissions(manifest, menu.ID)
	if err != nil {
		return nil, err
	}

	result := &ModuleInstallResult{
		MenuID:        menu.ID,
		MenuSlug:      menu.Slug,
		PermissionIDs: permissionIDs,
	}

	manifestPath, err := i.SyncManifestFile(manifest)
	if err != nil {
		return nil, err
	}
	result.ManifestPath = manifestPath

	return result, nil
}

func (i *ModuleInstallerImpl) SyncManifestFile(manifest *ModuleManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is required")
	}
	if err := os.MkdirAll(generatedModuleManifestDir, 0755); err != nil {
		return "", fmt.Errorf("create manifest dir failed: %w", err)
	}

	filePath := filepath.Join(generatedModuleManifestDir, manifest.ModuleName+".json")
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal manifest failed: %w", err)
	}
	if err := os.WriteFile(filePath, payload, 0644); err != nil {
		return "", fmt.Errorf("write manifest failed: %w", err)
	}
	return filePath, nil
}

func (i *ModuleInstallerImpl) upsertMenu(manifest *ModuleManifest) (*models.Menu, error) {
	var parentID uint
	if manifest.ParentMenuSlug != "" {
		var parent models.Menu
		if err := appfacades.OrmQuery(i.ctx).Where("slug", manifest.ParentMenuSlug).First(&parent); err != nil || parent.ID == 0 {
			return nil, apperrors.ErrMenuNotFound.WithParams(map[string]any{"slug": manifest.ParentMenuSlug})
		}
		parentID = parent.ID
	}

	var existing models.Menu
	err := appfacades.OrmQuery(i.ctx).Where("slug", manifest.MenuSlug).First(&existing)
	if err == nil && existing.ID > 0 {
		updates := map[string]any{
			"title":     manifest.MenuTitle,
			"path":      manifest.Path,
			"component": manifest.Component,
			"icon":      manifest.Icon,
			"sort":      manifest.MenuSort,
			"type":      uint8(2),
			"status":    uint8(1),
			"parent_id": parentID,
		}
		if _, err := appfacades.OrmQuery(i.ctx).Model(&models.Menu{}).Where("id", existing.ID).Update(updates); err != nil {
			return nil, err
		}
		if err := appfacades.OrmQuery(i.ctx).Where("id", existing.ID).First(&existing); err != nil {
			return nil, err
		}
		return &existing, nil
	}

	menu := models.Menu{
		ParentID:  parentID,
		Title:     manifest.MenuTitle,
		Slug:      manifest.MenuSlug,
		Icon:      manifest.Icon,
		Path:      manifest.Path,
		Component: manifest.Component,
		Type:      2,
		Status:    1,
		Sort:      manifest.MenuSort,
		LinkType:  1,
		OpenType:  1,
	}
	if err := appfacades.OrmQuery(i.ctx).Create(&menu); err != nil {
		return nil, err
	}
	return &menu, nil
}

func (i *ModuleInstallerImpl) upsertPermissions(manifest *ModuleManifest, menuID uint) ([]uint, error) {
	if menuID == 0 {
		return nil, fmt.Errorf("menu_id is required")
	}

	var permissionIDs []uint
	for _, def := range manifest.Permissions {
		if strings.TrimSpace(def.Slug) == "" {
			continue
		}

		perm := models.Permission{
			Name:        def.Name,
			Slug:        def.Slug,
			Method:      def.Method,
			Path:        def.Path,
			Description: def.Description,
			Status:      1,
			Sort:        def.Sort,
			MenuID:      menuID,
		}

		var existing models.Permission
		if err := appfacades.OrmQuery(i.ctx).Where("slug", def.Slug).First(&existing); err == nil && existing.ID > 0 {
			if _, err := appfacades.OrmQuery(i.ctx).Model(&models.Permission{}).Where("id", existing.ID).Update(map[string]any{
				"name":        def.Name,
				"method":      def.Method,
				"path":        def.Path,
				"description": def.Description,
				"status":      1,
				"sort":        def.Sort,
				"menu_id":     menuID,
			}); err != nil {
				return nil, err
			}
			permissionIDs = append(permissionIDs, existing.ID)
			continue
		}

		if err := appfacades.OrmQuery(i.ctx).Create(&perm); err != nil {
			return nil, err
		}
		permissionIDs = append(permissionIDs, perm.ID)
	}

	return permissionIDs, nil
}

// LoadGeneratedModuleManifests reads all JSON manifests written by the code generator.
func LoadGeneratedModuleManifests() ([]ModuleManifest, error) {
	entries, err := os.ReadDir(generatedModuleManifestDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var manifests []ModuleManifest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(generatedModuleManifestDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var manifest ModuleManifest
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, fmt.Errorf("parse manifest %s failed: %w", entry.Name(), err)
		}
		if manifest.ModuleName == "" {
			continue
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

// InstallGeneratedModuleManifests installs all persisted module manifests (used by seeder).
func InstallGeneratedModuleManifests(ctx context.Context) error {
	manifests, err := LoadGeneratedModuleManifests()
	if err != nil {
		return err
	}
	installer := NewModuleInstaller(ctx)
	for _, manifest := range manifests {
		copyManifest := manifest
		if _, err := installer.Install(&copyManifest); err != nil {
			facades.Log().Errorf("install generated module %s failed: %v", manifest.ModuleName, err)
			return err
		}
	}
	return nil
}
