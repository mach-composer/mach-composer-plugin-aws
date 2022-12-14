package internal

import (
	"fmt"

	"github.com/creasty/defaults"
	"github.com/elliotchance/pie/v2"
	"github.com/mach-composer/mach-composer-plugin-helpers/helpers"
	"github.com/mach-composer/mach-composer-plugin-sdk/plugin"
	"github.com/mach-composer/mach-composer-plugin-sdk/schema"
	"github.com/mitchellh/mapstructure"
)

func NewAWSPlugin() schema.MachComposerPlugin {
	state := &Plugin{
		provider:         "3.74.1",
		siteConfigs:      map[string]SiteConfig{},
		componentConfigs: map[string]ComponentConfig{},
		endpointsConfigs: map[string]map[string]EndpointConfig{},
	}

	return plugin.NewPlugin(&schema.PluginSchema{
		Identifier: "aws",

		Configure: state.Configure,
		IsEnabled: state.IsEnabled,

		// Config
		SetRemoteStateBackend: state.SetRemoteStateBackend,
		SetSiteConfig:         state.SetSiteConfig,

		// Schema
		GetValidationSchema: state.GetValidationSchema,

		// Config endpoints
		SetSiteEndpointConfig:       state.SetSiteEndpointConfig,
		SetComponentEndpointsConfig: state.SetComponentEndpointsConfig,

		// Renders
		RenderTerraformStateBackend: state.TerraformRenderStateBackend,
		RenderTerraformProviders:    state.TerraformRenderProviders,
		RenderTerraformResources:    state.TerraformRenderResources,
		RenderTerraformComponent:    state.RenderTerraformComponent,
	})
}

type Plugin struct {
	environment      string
	provider         string
	remoteState      *AWSTFState
	siteConfigs      map[string]SiteConfig
	componentConfigs map[string]ComponentConfig
	endpointsConfigs map[string]map[string]EndpointConfig
}

func (p *Plugin) Configure(environment string, provider string) error {
	p.environment = environment
	if provider != "" {
		p.provider = provider
	}
	return nil
}

func (p *Plugin) IsEnabled() bool {
	return len(p.siteConfigs) > 0
}

func (p *Plugin) Identifier() string {
	return "aws"
}

func (p *Plugin) GetValidationSchema() (*schema.ValidationSchema, error) {
	result := getSchema()
	return result, nil
}

func (p *Plugin) SetRemoteStateBackend(data map[string]any) error {
	state := &AWSTFState{}
	if err := mapstructure.Decode(data, state); err != nil {
		return err
	}
	if err := defaults.Set(state); err != nil {
		return err
	}
	p.remoteState = state
	return nil
}

func (p *Plugin) SetSiteConfig(site string, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	cfg := SiteConfig{}
	if val, ok := data["account_id"].(int); ok {
		data["account_id"] = fmt.Sprintf("%d", val)
	}

	if err := mapstructure.Decode(data, &cfg); err != nil {
		return err
	}

	if err := defaults.Set(&cfg); err != nil {
		return err
	}

	p.siteConfigs[site] = cfg
	return nil
}

func (p *Plugin) SetSiteEndpointConfig(site string, name string, data map[string]any) error {
	configs, ok := p.endpointsConfigs[site]
	if !ok {
		configs = map[string]EndpointConfig{}
		p.endpointsConfigs[site] = configs
	}

	cfg := EndpointConfig{
		Key: name,
	}

	if err := mapstructure.Decode(data, &cfg); err != nil {
		return err
	}

	if err := defaults.Set(&cfg); err != nil {
		return err
	}

	configs[name] = cfg
	p.endpointsConfigs[site] = configs
	return nil
}

func (p *Plugin) SetComponentEndpointsConfig(component string, endpoints map[string]string) error {
	cfg, ok := p.componentConfigs[component]
	if !ok {
		cfg = ComponentConfig{}
		p.componentConfigs[component] = cfg
	}
	cfg.Endpoints = endpoints
	p.componentConfigs[component] = cfg
	return nil
}

func (p *Plugin) TerraformRenderStateBackend(site string) (string, error) {
	if p.remoteState == nil {
		return "", nil
	}

	templateContext := struct {
		State *AWSTFState
		Site  string
	}{
		State: p.remoteState,
		Site:  site,
	}

	template := `
	backend "s3" {
	  bucket         = "{{ .State.Bucket }}"
	  key            = "{{ .State.KeyPrefix}}/{{ .Site }}"
	  region         = "{{ .State.Region }}"
	  {{ if .State.RoleARN }}
	  role_arn       = "{{ .State.RoleARN }}"
	  {{ end }}
	  {{ if .State.LockTable }}
	  dynamodb_table = "{{ .State.LockTable }}"
	  {{ end }}
	  encrypt        = {{ .State.Encrypt }}
	}
	`
	return helpers.RenderGoTemplate(template, templateContext)
}

func (p *Plugin) TerraformRenderProviders(site string) (string, error) {
	cfg := p.getSiteConfig(site)
	if cfg == nil {
		return "", nil
	}

	result := fmt.Sprintf(`
		aws = {
			version = "%s"
		}`, helpers.VersionConstraint(p.provider))
	return result, nil
}

func (p *Plugin) TerraformRenderResources(site string) (string, error) {
	cfg := p.getSiteConfig(site)
	if cfg == nil {
		return "", nil
	}

	activeEndpoints := map[string]EndpointConfig{}
	siteEndpoint := p.endpointsConfigs[site]

	needsDefaultEndpoint := false
	for _, component := range p.componentConfigs {
		for _, external := range component.Endpoints {
			if external == "default" {
				needsDefaultEndpoint = true
			}

			endpointConfig, ok := siteEndpoint[external]
			if !ok && external != "default" {
				return "", fmt.Errorf("component requires undeclared endpoint: %s", external)
			}

			if _, ok := activeEndpoints[external]; !ok {
				activeEndpoints[external] = endpointConfig
			}
		}
	}

	if needsDefaultEndpoint {
		activeEndpoints["default"] = EndpointConfig{
			Key: "default",
		}
	}

	content, err := renderResources(site, p.environment, cfg, pie.Values(activeEndpoints))
	if err != nil {
		return "", fmt.Errorf("failed to render resources: %w", err)
	}

	return content, nil
}

func (p *Plugin) RenderTerraformComponent(site string, component string) (*schema.ComponentSchema, error) {
	cfg := p.getSiteConfig(site)
	if cfg == nil {
		return nil, nil
	}
	componentCfg := p.componentConfigs[component]

	result := &schema.ComponentSchema{
		DependsOn: terraformRenderComponentDependsOn(&componentCfg),
		Providers: TerraformRenderComponentProviders(cfg),
	}

	value, err := terraformRenderComponentVars(cfg, &componentCfg)
	if err != nil {
		return nil, err
	}
	result.Variables = value
	return result, nil
}

func (p *Plugin) getSiteConfig(site string) *SiteConfig {
	cfg, ok := p.siteConfigs[site]
	if !ok {
		return nil
	}
	return &cfg
}

func terraformRenderComponentVars(cfg *SiteConfig, componentCfg *ComponentConfig) (string, error) {
	endpointNames := map[string]string{}
	for key, value := range componentCfg.Endpoints {
		endpointNames[helpers.Slugify(key)] = helpers.Slugify(value)
	}

	templateContext := struct {
		Site      *SiteConfig
		Endpoints map[string]string
	}{
		Site:      cfg,
		Endpoints: endpointNames,
	}

	template := `
		{{ range $cEndpoint, $sEndpoint := .Endpoints }}
		aws_endpoint_{{ $cEndpoint }} = {
			url = local.endpoint_url_{{ $sEndpoint }}
			api_gateway_id = aws_apigatewayv2_api.{{ $sEndpoint }}_gateway.id
			api_gateway_execution_arn = aws_apigatewayv2_api.{{ $sEndpoint }}_gateway.execution_arn
		}
		{{ end }}`
	return helpers.RenderGoTemplate(template, templateContext)
}

func terraformRenderComponentDependsOn(componentCfg *ComponentConfig) []string {
	result := []string{}
	for _, value := range componentCfg.Endpoints {
		depends := fmt.Sprintf("aws_apigatewayv2_api.%s_gateway", helpers.Slugify(value))
		result = append(result, depends)
	}
	return result
}

func TerraformRenderComponentProviders(cfg *SiteConfig) []string {
	providers := []string{"aws = aws"}
	for _, provider := range cfg.ExtraProviders {
		providers = append(providers, fmt.Sprintf("aws.%s = aws.%s", provider.Name, provider.Name))
	}
	return providers
}
