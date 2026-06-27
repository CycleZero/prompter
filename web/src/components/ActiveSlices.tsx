import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Box, Typography, IconButton, TextField } from '@mui/material';
import { Delete, DragIndicator } from '@mui/icons-material';
import { usePromptStore } from '../store';
import type { ActiveSlice } from '../types';

// 可拖拽排序的单个片段行
function SortableSlice({
  regionId,
  slice,
}: {
  regionId: number;
  slice: ActiveSlice;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ id: `${regionId}-${slice.slice_id}` });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const updateText = usePromptStore((s) => s.updateSliceCustomText);
  const removeSlice = usePromptStore((s) => s.removeSliceFromRegion);

  return (
    <Box
      ref={setNodeRef}
      style={style}
      sx={{ display: 'flex', alignItems: 'center', gap: 0.5, py: 0.5 }}
    >
      {/* 拖拽手柄 */}
      <Box {...attributes} {...listeners} sx={{ cursor: 'grab' }}>
        <DragIndicator fontSize="small" />
      </Box>

      {/* 可编辑的自定义文本 */}
      <TextField
        size="small"
        variant="standard"
        value={slice.custom_text ?? ''}
        onChange={(e) =>
          updateText(regionId, slice.slice_id, e.target.value || null)
        }
        placeholder={`Slice #${slice.slice_id}`}
        sx={{ flexGrow: 1 }}
      />

      {/* 删除按钮 */}
      <IconButton
        size="small"
        onClick={() => removeSlice(regionId, slice.slice_id)}
      >
        <Delete fontSize="small" />
      </IconButton>
    </Box>
  );
}

/** 已选片段列表 — 按区域分组展示，支持拖拽排序 */
export function ActiveSlices() {
  const regions = usePromptStore((s) => s.regions);
  const moveSlice = usePromptStore((s) => s.moveSlice);

  // 拖拽传感器：需要移动 5px 后才会触发拖拽，避免误触
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  /** 拖拽结束回调 */
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const regionId = Number(String(active.id).split('-')[0]);
    const region = regions.find((r) => r.region_id === regionId);
    if (!region) return;

    const oldIndex = region.slices.findIndex(
      (s) => `${regionId}-${s.slice_id}` === active.id,
    );
    const newIndex = region.slices.findIndex(
      (s) => `${regionId}-${s.slice_id}` === over.id,
    );
    if (oldIndex !== -1 && newIndex !== -1) {
      moveSlice(regionId, oldIndex, newIndex);
    }
  };

  return (
    <>
      {regions.map((r) => (
        <Box key={r.region_id} sx={{ mb: 2 }}>
          {/* 区域名称 */}
          <Typography variant="subtitle2" gutterBottom>
            {r.region_name}
          </Typography>

          {/* 可拖拽排序的片段列表 */}
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={r.slices.map((s) => `${r.region_id}-${s.slice_id}`)}
              strategy={verticalListSortingStrategy}
            >
              {r.slices.map((s) => (
                <SortableSlice
                  key={`${r.region_id}-${s.slice_id}`}
                  regionId={r.region_id}
                  slice={s}
                />
              ))}
            </SortableContext>
          </DndContext>
        </Box>
      ))}
    </>
  );
}
