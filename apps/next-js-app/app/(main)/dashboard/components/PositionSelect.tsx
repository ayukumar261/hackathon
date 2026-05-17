"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { useSelectedPosition } from "@/lib/hooks/useSelectedPosition";

export function PositionSelect() {
  const { positions, selectedId, setSelectedId, isLoading } =
    useSelectedPosition();

  if (isLoading) {
    return (
      <Select disabled>
        <SelectTrigger aria-label="Position" className="text-muted-foreground">
          <span className="flex items-center gap-2">
            <Spinner className="size-4 text-muted-foreground" />
            <span>Loading positions…</span>
          </span>
        </SelectTrigger>
      </Select>
    );
  }

  if (positions.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">No positions yet.</div>
    );
  }

  return (
    <Select
      value={selectedId ?? undefined}
      onValueChange={(value) => setSelectedId(value)}
    >
      <SelectTrigger aria-label="Position">
        <SelectValue placeholder="Select a position…" />
      </SelectTrigger>
      <SelectContent>
        {positions.map((position) => (
          <SelectItem key={position.id} value={position.id}>
            <span className="truncate">{position.title}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export default PositionSelect;
