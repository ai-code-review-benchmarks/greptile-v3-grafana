import { useCallback, useEffect, useMemo, useState } from 'react';

import { isBitbucketResponse, isGitHubResponse, isGitLabResponse, isSupportedGitProvider } from '../guards';
import {
  GitProviderApiResponse,
  RepositoryOption,
  UseRepositoryFetchingProps,
  UseRepositoryFetchingResult,
} from '../types/repository';
import { createApiRequest, getErrorMessage, makeApiRequest } from '../utils/httpUtils';

export function useRepositoryFetching({
  repositoryType,
  repositoryToken,
}: UseRepositoryFetchingProps): UseRepositoryFetchingResult {
  const [repositories, setRepositories] = useState<RepositoryOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const hasRequiredData = useMemo(() => {
    if (!isSupportedGitProvider(repositoryType)) {
      return false;
    }

    return repositoryToken && repositoryToken.trim().length > 0;
  }, [repositoryType, repositoryToken]);

  const fetchRepositories = useCallback(async () => {
    if (!hasRequiredData) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const apiConfig = createApiRequest(repositoryType, repositoryToken);
      let repositoryData: RepositoryOption[] = [];

      const data: GitProviderApiResponse = await makeApiRequest(apiConfig.repositories);

      if (isGitHubResponse(data)) {
        repositoryData = data.map((repo) => ({
          fullName: repo.full_name,
          url: repo.clone_url,
        }));
      } else if (isGitLabResponse(data)) {
        repositoryData = data.map((project) => ({
          fullName: project.path_with_namespace,
          url: project.http_url_to_repo,
        }));
      } else if (isBitbucketResponse(data)) {
        repositoryData = data.values.map((repo) => ({
          fullName: repo.full_name,
          url: repo.links.clone.find((link) => link.name === 'https')?.href || '',
        }));
      }

      setRepositories(repositoryData);
    } catch (err: unknown) {
      console.error('Failed to fetch repositories:', err);
      setError(getErrorMessage(err));
      setRepositories([]);
    } finally {
      setLoading(false);
    }
  }, [hasRequiredData, repositoryToken, repositoryType]);

  useEffect(() => {
    if (hasRequiredData) {
      setError(null);
      const timeoutId = setTimeout(fetchRepositories, 500);
      return () => clearTimeout(timeoutId);
    } else {
      setRepositories([]);
      setError(null);
      setLoading(false);
      return undefined;
    }
  }, [hasRequiredData, fetchRepositories, repositoryType, repositoryToken]);

  return {
    repositories,
    loading,
    error,
  };
}
