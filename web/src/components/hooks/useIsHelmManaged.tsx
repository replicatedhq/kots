import { useQuery, UseQueryResult } from "react-query";

interface IsHelmManagedResponse {
  isHelmManaged: boolean;
}

type IsHelmManaged = boolean;

async function fetchIsHelmManaged({
  apiEndpoint = process.env.API_ENDPOINT,
} = {}): Promise<IsHelmManagedResponse> {
  try {
    const res = await fetch(`${apiEndpoint}/is-helm-managed`, {
      headers: {
        "Content-Type": "application/json",
      },
      method: "GET",
      credentials: "include",
    });
    if (res.ok) {
      return await res.json();
    }
    throw new Error("Error fetching isHelmManaged");
  } catch (err) {
    if (err instanceof Error)
      throw Error(err?.message || "Error fetching isHelmManaged");
    else throw Error("Error fetching isHelmManaged");
  }
}

function useIsHelmManaged() {
  return useQuery({
    queryFn: () => fetchIsHelmManaged(),
    queryKey: "isHelmManaged",
    staleTime: Infinity,
    select: (response): IsHelmManaged => response.isHelmManaged || false,
  });
}

function UseIsHelmManaged({
  children,
}: {
  children: (props: UseQueryResult) => JSX.Element;
}) {
  const query = useIsHelmManaged();

  return children(query);
}

export { UseIsHelmManaged, fetchIsHelmManaged, useIsHelmManaged };
