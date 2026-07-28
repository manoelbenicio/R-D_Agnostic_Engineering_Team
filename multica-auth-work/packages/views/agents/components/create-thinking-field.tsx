"use client";

import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { runtimeModelsOptions } from "@multica/core/runtimes";
import type { RuntimeModel } from "@multica/core/types";
import { Label } from "@multica/ui/components/ui/label";
import { useT } from "../../i18n";
import { ThinkingPicker } from "./inspector/thinking-picker";

/**
 * Reasoning selector for Create/Duplicate Agent.
 *
 * Structured catalogs (Kiro/Codex) expose `thinking.supported_levels` and
 * render this independent picker. AGY encodes its tier in the model ID and
 * therefore has no `thinking` block; in that case the field stays hidden and
 * any stale structured value is cleared once discovery completes.
 */
export function CreateThinkingField({
  runtimeId,
  runtimeOnline,
  model,
  value,
  onChange,
}: {
  runtimeId: string | null;
  runtimeOnline: boolean;
  model: string;
  value: string;
  onChange: (next: string) => void;
}) {
  const { t } = useT("agents");
  const modelsQuery = useQuery(
    runtimeModelsOptions(runtimeOnline ? runtimeId : null),
  );
  const levels = useMemo(() => {
    const models = modelsQuery.data?.models ?? [];
    return pickModelEntry(models, model)?.thinking?.supported_levels ?? [];
  }, [model, modelsQuery.data?.models]);
  const catalogReady = modelsQuery.isSuccess;

  useEffect(() => {
    if (
      catalogReady &&
      value &&
      !levels.some((level) => level.value === value)
    ) {
      onChange("");
    }
  }, [catalogReady, levels, onChange, value]);

  if (levels.length === 0) return null;

  return (
    <div className="flex min-w-0 flex-col">
      <Label className="text-xs text-muted-foreground">
        {t(($) => $.inspector.prop_thinking)}
      </Label>
      <div className="mt-1.5">
        <ThinkingPicker
          value={value}
          levels={levels}
          variant="field"
          onChange={onChange}
        />
      </div>
    </div>
  );
}

function pickModelEntry(
  models: RuntimeModel[],
  model: string,
): RuntimeModel | undefined {
  if (model) return models.find((entry) => entry.id === model);
  return models.find((entry) => entry.default) ?? models[0];
}
