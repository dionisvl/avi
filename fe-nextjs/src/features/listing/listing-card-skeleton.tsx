import { Skeleton } from "@/components/ui/skeleton";

export function ListingCardSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <Skeleton className="aspect-[1.58/1] w-full rounded-[14px]" />

      <div className="flex flex-1 flex-col gap-2 pt-3">
        <Skeleton className="h-4 w-full rounded-md" />
        <Skeleton className="h-4 w-3/4 rounded-md" />
        <Skeleton className="mt-1 h-4 w-1/2 rounded-md" />
        <Skeleton className="mt-1 h-3 w-2/3 rounded-md" />
      </div>
    </div>
  );
}
