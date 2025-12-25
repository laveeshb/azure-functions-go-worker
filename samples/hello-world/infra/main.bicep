targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the environment (used to generate resource names)')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Name of the resource group')
param resourceGroupName string = ''

@description('Name of the Function App')
param functionAppName string = ''

@description('Name of the Storage Account')
param storageAccountName string = ''

@description('Name of the App Service Plan')
param appServicePlanName string = ''

var abbrs = loadJsonContent('./abbreviations.json')
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))
var tags = {
  'azd-env-name': environmentName
}

// Resource Group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// Function App infrastructure
module functionApp './modules/function-app.bicep' = {
  name: 'functionApp'
  scope: rg
  params: {
    location: location
    tags: tags
    functionAppName: !empty(functionAppName) ? functionAppName : '${abbrs.webSitesFunctions}${resourceToken}'
    storageAccountName: !empty(storageAccountName) ? storageAccountName : '${abbrs.storageStorageAccounts}${resourceToken}'
    appServicePlanName: !empty(appServicePlanName) ? appServicePlanName : '${abbrs.webServerFarms}${resourceToken}'
  }
}

output AZURE_LOCATION string = location
output AZURE_RESOURCE_GROUP string = rg.name
output AZURE_FUNCTION_APP_NAME string = functionApp.outputs.functionAppName
output AZURE_FUNCTION_APP_URL string = functionApp.outputs.functionAppUrl
