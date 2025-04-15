import { execSync } from 'child_process';
import path from 'path';

// Custom type for plop action functions
export type PlopActionFunction = (
  answers: Record<string, unknown>, 
  config?: Record<string, unknown>
) => string | Promise<string>;

// Helper function to create paths relative to project root
export const projectPath = (basePath: string) => (relativePath: string): string => 
  path.join(basePath, relativePath);

// Helper to remove quotes from operation IDs
export const removeQuotes = (str: string | unknown): string | unknown => {
  if (typeof str !== 'string') {
    return str;
  }
  return str.replace(/^['"](.*)['"]$/, '$1');
};

// Helper to format operation IDs for filter endpoints
export const formatOperationIds = () => (operationArray: string[]): string => {
  if (!Array.isArray(operationArray)) {
    return '';
  }
  return operationArray.map((op) => `'${removeQuotes(op)}'`).join(', ');
};

// List of created or modified files
export const getFilesToFormat = (groupName: string): string[] => [
  `public/app/api/clients/${groupName}/baseAPI.ts`,
  `public/app/api/clients/${groupName}/index.ts`,
  `scripts/generate-rtk-apis.ts`,
  `public/app/core/reducers/root.ts`,
  `public/app/store/configureStore.ts`,
];

// Action function for running yarn generate-apis
export const runGenerateApis = (basePath: string): PlopActionFunction => (_, __) => {
  try {
    console.log('⏳ Running yarn generate-apis to generate endpoints...');
    execSync('yarn generate-apis', { stdio: 'inherit', cwd: basePath });
    return '✅ API endpoints generated successfully!';
  } catch (error) {
    if (error instanceof Error) {
      console.error('❌ Failed to generate API endpoints:', error.message);
    } else {
      console.error('❌ Failed to generate API endpoints:', String(error));
    }
    return '❌ Failed to generate API endpoints. See error above.';
  }
};

// Action function for formatting files with prettier and eslint
export const formatFiles = (
  basePath: string, 
  createProjectPath: ReturnType<typeof projectPath>
): PlopActionFunction => (_, config) => {
  // Ensure config is present and has the expected shape
  if (!config || !Array.isArray(config.files)) {
    console.error('Invalid config passed to formatFiles action');
    return '❌ Formatting failed: Invalid configuration';
  }

  const filesToFormat = config.files.map((file: string) => createProjectPath(file));
  
  try {
    const filesList = filesToFormat.map((file: string) => `"${file}"`).join(' ');

    console.log('🧹 Running ESLint on generated/modified files...');
    try {
      execSync(`yarn eslint --fix ${filesList}`, { cwd: basePath });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.warn(`⚠️ Warning: ESLint encountered issues: ${errorMessage}`);
    }

    console.log('🧹 Running Prettier on generated/modified files...');
    try {
      execSync(`yarn prettier --write ${filesList}`, { cwd: basePath });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      console.warn(`⚠️ Warning: Prettier encountered issues: ${errorMessage}`);
    }
    
    return '✅ Files linted and formatted successfully!';
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error('⚠️ Warning: Formatting operations failed:', errorMessage);
    return '⚠️ Warning: Formatting operations failed.';
  }
};

// Validation helpers
export const validateGroup = (group: string): boolean | string => {
  return group && group.includes('.grafana.app') ? true : 'Group should be in format: name.grafana.app';
};

export const validateVersion = (version: string): boolean | string => {
  return version && /^v\d+[a-z]*\d+$/.test(version) ? true : 'Version should be in format: v0alpha1, v1beta2, etc.';
}; 
