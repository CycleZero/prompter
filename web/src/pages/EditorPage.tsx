import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  TextField,
  Paper,
  Typography,
  IconButton,
  Chip,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import { Save, Close, Add, DragIndicator } from '@mui/icons-material';
import {
  DndContext,
  DragOverlay,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  SortableContext,
  horizontalListSortingStrategy,
  rectSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { api } from '../api/client';
import { usePromptStore } from '../store';
import type { ActiveSlice, SliceType, Slice, SearchSlice } from '../types';
import { RegionPanel } from '../components/RegionPanel';

// ============================================================
// SortableSlice — 可拖拽排序的片段标签
// ============================================================

interface SortableSliceProps {
  slice: ActiveSlice;
  regionId: number;
  onRemove: (regionId: number, sliceId: number) => void;
}

/** 可拖拽排序的片段 Chip 组件 */
function SortableSlice({ slice, regionId, onRemove }: SortableSliceProps) {
  const { setNodeRef, transform, transition, listeners, attributes } =
    useSortable({ id: `${regionId}-slice-${slice.slice_id}` });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: transform ? 0.3 : 1,
    zIndex: transform ? 10 : 'auto',
  };

  return (
    <Box ref={setNodeRef} style={style} {...listeners} {...attributes}>
      <Chip
        label={slice.custom_text ?? (slice.translated_content || slice.content || `#${slice.slice_id}`)}
        title={slice.translated_content ? slice.content : undefined}
        size="medium"
        color="primary"
        variant="filled"
        onDelete={() => onRemove(regionId, slice.slice_id)}
        sx={{ fontSize: '0.85rem', cursor: 'grab', py: 0.5 }}
      />
    </Box>
  );
}

// ============================================================
// SortableRegion — 可拖拽排序的区域分组框
// ============================================================

interface SortableRegionProps {
  region: { region_id: number; region_name: string; slices: ActiveSlice[] };
  onRemoveRegion: (regionId: number) => void;
  onRemoveSlice: (regionId: number, sliceId: number) => void;
  onRegionNameChange: (regionId: number, name: string) => void;
}

/** 可拖拽的 Region 分组框 — 含拖拽手柄、可编辑标题、内部片段排序 */
function SortableRegion({
  region,
  onRemoveRegion,
  onRemoveSlice,
  onRegionNameChange,
}: SortableRegionProps) {
  const {
    setNodeRef,
    transform,
    transition,
    listeners,
    attributes,
  } = useSortable({ id: `region-${region.region_id}` });

  // 区域名称内联编辑状态
  const [editingName, setEditingName] = useState(false);
  const [nameValue, setNameValue] = useState(region.region_name);

  const sliceIds = region.slices.map((s) => `${region.region_id}-slice-${s.slice_id}`);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: transform ? 1 : undefined,
  };

  /** 提交区域名称修改 */
  const commitName = () => {
    const trimmed = nameValue.trim();
    if (trimmed && trimmed !== region.region_name) {
      onRegionNameChange(region.region_id, trimmed);
    } else {
      setNameValue(region.region_name);
    }
    setEditingName(false);
  };

  return (
    <Box
      ref={setNodeRef}
      style={style}
      sx={{
        border: 1,
        borderColor: 'divider',
        borderRadius: 1,
        p: 2,
        minWidth: 220,
        bgcolor: 'white',
        boxShadow: transform ? 4 : 0,
      }}
    >
      {/* 标题行：拖拽手柄 + 可编辑标题 + 删除按钮 */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
        <Box
          {...listeners}
          {...attributes}
          sx={{ display: 'flex', alignItems: 'center', cursor: 'grab', mr: 0.5 }}
        >
          <DragIndicator fontSize="small" sx={{ color: 'text.secondary' }} />
        </Box>
        {editingName ? (
          <TextField
            size="small"
            variant="standard"
            value={nameValue}
            onChange={(e) => setNameValue(e.target.value)}
            onBlur={commitName}
            onKeyDown={(e) => {
              if (e.key === 'Enter') commitName();
              if (e.key === 'Escape') {
                setNameValue(region.region_name);
                setEditingName(false);
              }
            }}
            autoFocus
            sx={{ flexGrow: 1, mr: 0.5 }}
            slotProps={{
              input: { sx: { fontWeight: 600, fontSize: '0.875rem' } },
            }}
          />
        ) : (
          <Typography
            variant="subtitle2"
            sx={{
              fontWeight: 600,
              flexGrow: 1,
              cursor: 'pointer',
              '&:hover': { color: 'primary.main' },
            }}
            onClick={() => {
              setNameValue(region.region_name);
              setEditingName(true);
            }}
          >
            {region.region_name}
          </Typography>
        )}
        <IconButton
          size="small"
          onClick={() => onRemoveRegion(region.region_id)}
          sx={{ flexShrink: 0 }}
        >
          <Close fontSize="inherit" />
        </IconButton>
      </Box>

      {/* 片段列表：SortableContext（无嵌套 DndContext，由外层统一管理） */}
      <SortableContext items={sliceIds} strategy={rectSortingStrategy}>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
          {region.slices.length === 0 ? (
            <Typography variant="caption" color="text.disabled" sx={{ py: 0.5 }}>
              暂无标签，从下方提示词库中添加
            </Typography>
          ) : (
            region.slices.map((slice) => (
              <SortableSlice
                key={slice.slice_id}
                slice={slice}
                regionId={region.region_id}
                onRemove={onRemoveSlice}
              />
            ))
          )}
        </Box>
      </SortableContext>
    </Box>
  );
}

// ============================================================
// EditorPage — 编辑器主页面
// ============================================================

/** 编辑器页面 — 单页布局：当前 Prompt 区域 + 提示词库 + 底部预览栏 */
export function EditorPage() {
  // 提示词类型树（分类数据）
  const [sliceTypes, setSliceTypes] = useState<SliceType[]>([]);
  // 保存状态提示
  const [status, setStatus] = useState('');
  // 当前添加标签的目标 Region ID（0 = 自动选择第一个或创建）
  const [targetRegionId, setTargetRegionId] = useState(0);
  // Zustand 全局状态
  const { title, setTitle, regions, getPromptPreview } = usePromptStore();
  // 当前活跃的拖拽项 ID（用于 DragOverlay）
  const [activeId, setActiveId] = useState<string | null>(null);

  // 拖拽传感器配置：5px 移动阈值防止误触
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  // 初始化：加载片段类型树和活动 Prompt 数据
  useEffect(() => {
    api.getSliceTypes()
      .then((res) => setSliceTypes(res.data.types))
      .catch(() => console.error('加载类型树失败'));
    api.getActivePrompt().then((res) => {
      if (res.data.regions?.length) {
        setTitle(res.data.title || '');
        usePromptStore.getState().setRegions(res.data.regions);
      }
    }).catch(() => {});
  }, []);

  // ==========================================================
  // 提示词库标签点击处理
  // ==========================================================

  /** 
   * 点击提示词库中的标签 → 添加到当前选中的目标 Region
   * 防止同一 Region 内重复添加相同 slice_id
   */
  const handleSliceClick = (_typeName: string, slice: Slice | SearchSlice) => {
    const currentRegions = usePromptStore.getState().regions;
    // 确定目标 Region ID
    let targetId = targetRegionId;
    if (targetId === 0 && currentRegions.length > 0) {
      targetId = currentRegions[0].region_id;
    }
    // 如果没有目标 Region，创建默认「活跃」
    if (targetId === 0 || !currentRegions.find(r => r.region_id === targetId)) {
      targetId = Date.now();
      usePromptStore.getState().addRegion({
        region_id: targetId, region_name: '活跃', sort_order: currentRegions.length,
        slices: [{
          slice_id: slice.id, content: slice.content,
          translated_content: (slice as Slice).translated_content ?? '',
          sort_order: 0, custom_text: null,
        }],
      });
      setTargetRegionId(targetId);
      return;
    }
    // 检查重复：同一 Region 内不允许相同 slice_id
    const targetRegion = currentRegions.find(r => r.region_id === targetId)!;
    if (targetRegion.slices.some(s => s.slice_id === slice.id)) {
      return; // 已存在，忽略
    }
    usePromptStore.getState().addSliceToRegion(targetId, {
      slice_id: slice.id, content: slice.content,
      translated_content: (slice as Slice).translated_content ?? '',
      sort_order: targetRegion.slices.length, custom_text: null,
    });
    setTargetRegionId(targetId);
  };

  // ==========================================================
  // Region 操作
  // ==========================================================

  /** 添加新空白 Region */
  const handleAddRegion = () => {
    const currentRegions = usePromptStore.getState().regions;
    usePromptStore.getState().addRegion({
      region_id: Date.now(),
      region_name: '新分组',
      sort_order: currentRegions.length,
      slices: [],
    });
  };

  /** 删除整个 Region */
  const handleRemoveRegion = (regionId: number) => {
    usePromptStore.getState().removeRegion(regionId);
  };

  /** 更新 Region 名称 */
  const handleRegionNameChange = (regionId: number, name: string) => {
    const currentRegions = usePromptStore.getState().regions;
    usePromptStore.getState().setRegions(
      currentRegions.map((r) =>
        r.region_id === regionId ? { ...r, region_name: name } : r,
      ),
    );
  };

  // ==========================================================
  // 片段操作
  // ==========================================================

  /** 从 Region 中移除指定片段 */
  const handleRemoveSlice = (regionId: number, sliceId: number) => {
    usePromptStore.getState().removeSliceFromRegion(regionId, sliceId);
  };

  // ==========================================================
  // 拖拽排序处理
  // ==========================================================

  /** 拖拽中保持拖拽项原始位置不动（使用 DragOverlay 渲染副本） */
  const handleDragStart = (event: { active: { id: string | number } }) => {
    setActiveId(String(event.active.id));
  };

  /** 统一的拖拽处理器 — 根据 active.id 前缀分派 Region 或 Slice 操作 */
  const handleDragEnd = (event: DragEndEvent) => {
    setActiveId(null);
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeId = String(active.id);
    const currentRegions = usePromptStore.getState().regions;

    // Region 拖拽
    if (activeId.startsWith('region-')) {
      const oldIndex = currentRegions.findIndex(r => `region-${r.region_id}` === activeId);
      const newIndex = currentRegions.findIndex(r => `region-${r.region_id}` === String(over.id));
      if (oldIndex !== -1 && newIndex !== -1) {
        usePromptStore.getState().moveRegion(oldIndex, newIndex);
      }
      return;
    }

    // Slice 拖拽（可能跨 Region）
    const srcRegionId = parseInt(activeId.split('-slice-')[0]);
    const srcRegion = currentRegions.find(r => r.region_id === srcRegionId);
    if (!srcRegion) return;
    const srcIndex = srcRegion.slices.findIndex(s => `${srcRegionId}-slice-${s.slice_id}` === activeId);
    if (srcIndex === -1) return;

    // 判断目标 Region
    const overId = String(over.id);
    const dstRegionId = parseInt(overId.split('-slice-')[0]);
    const dstRegion = currentRegions.find(r => r.region_id === dstRegionId);

    if (!dstRegion) {
      // 拖到 Region 上或空区域，不处理（回原位）
      return;
    }

    // 判断目标是另一个 Region 的哪个位置
    const dstIndex = dstRegion.slices.findIndex(s => `${dstRegionId}-slice-${s.slice_id}` === overId);

    if (srcRegionId === dstRegionId) {
      // 同一 Region 内移动
      usePromptStore.getState().moveSlice(srcRegionId, srcIndex, dstIndex !== -1 ? dstIndex : srcIndex);
    } else {
      // 跨 Region 移动：从源 Region 移除 → 添加到目标 Region
      const slice = srcRegion.slices[srcIndex];
      usePromptStore.getState().removeSliceFromRegion(srcRegionId, slice.slice_id);
      const targetPos = dstIndex !== -1 ? dstIndex : dstRegion.slices.length;
      usePromptStore.getState().addSliceToRegion(dstRegionId, {
        ...slice,
        sort_order: targetPos,
      });
    }
  };

  // ==========================================================
  // 保存
  // ==========================================================

  /** 保存当前 Prompt 到后端 */
  const handleSave = async () => {
    try {
      const currentRegions = usePromptStore.getState().regions;
      const currentTitle = usePromptStore.getState().title;
      await api.updateActivePrompt({
        title: currentTitle,
        regions: currentRegions,
        updated_at: '',
      });
      setStatus('已保存');
      setTimeout(() => setStatus(''), 2000);
    } catch {
      setStatus('保存失败');
    }
  };

  // ==========================================================
  // 计算属性
  // ==========================================================

  const preview = getPromptPreview();
  const regionIds = regions.map((r) => `region-${r.region_id}`);

  // ==========================================================
  // 渲染
  // ==========================================================

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flexGrow: 1, minHeight: 0 }}>
      {/* 标题栏：Prompt 标题 + 保存按钮 + 状态提示 */}
      <Box
        sx={{
          p: 2,
          display: 'flex',
          gap: 1,
          alignItems: 'center',
          flexShrink: 0,
        }}
      >
        <TextField
          size="small"
          label="Prompt 标题"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          sx={{ flexGrow: 1 }}
          slotProps={{
            input: {
              sx: { bgcolor: '#f8f9fa' },
            },
          }}
        />
        <Button
          variant="contained"
          startIcon={<Save />}
          onClick={handleSave}
          disabled={regions.length === 0}
        >
          保存
        </Button>
        {status && (
          <Chip
            label={status}
            color={status === '已保存' ? 'success' : 'error'}
            size="small"
          />
        )}
      </Box>

      {/* 主内容区域：当前 Prompt + 提示词库（双区域，可滚动） */}
      <Box sx={{ flexGrow: 1, overflow: 'auto', px: 2, pb: 1 }}>
        {/* 区域 1：当前 Prompt — 已选片段按 Region 分组展示，支持拖拽排序 */}
        <Paper sx={{ p: 2, mb: 2, bgcolor: '#fafbfc' }} variant="outlined">
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              mb: 1.5,
              gap: 1,
            }}
          >
            <Typography variant="subtitle1" sx={{ fontWeight: 600, flexGrow: 1 }}>
              当前 Prompt
            </Typography>
            <Button
              variant="outlined"
              size="small"
              startIcon={<Add />}
              onClick={handleAddRegion}
            >
              新建分组
            </Button>
            {regions.length > 0 && (
              <FormControl size="small" sx={{ minWidth: 120 }}>
                <InputLabel>添加到</InputLabel>
                <Select
                  value={targetRegionId || regions[0]?.region_id || ''}
                  label="添加到"
                  onChange={e => setTargetRegionId(Number(e.target.value))}
                >
                  {regions.map(r => (
                    <MenuItem key={r.region_id} value={r.region_id}>
                      {r.region_name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}
          </Box>

          {regions.length === 0 ? (
            <Typography color="text.secondary" variant="body2">
              点击下方标签库中的标签开始构建 Prompt
            </Typography>
          ) : (
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              onDragStart={handleDragStart}
              onDragEnd={handleDragEnd}
            >
              <SortableContext
                items={regionIds}
                strategy={horizontalListSortingStrategy}
              >
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2 }}>
                  {regions.map((region) => (
                    <SortableRegion
                      key={region.region_id}
                      region={region}
                      onRemoveRegion={handleRemoveRegion}
                      onRemoveSlice={handleRemoveSlice}
                      onRegionNameChange={handleRegionNameChange}
                    />
                  ))}
                </Box>
              </SortableContext>
              {/* 拖拽浮层：跟随鼠标的拖拽项副本 */}
              <DragOverlay dropAnimation={{ duration: 200, easing: 'ease' }}>
                {activeId && !activeId.startsWith('region-') ? (() => {
                  const [ridStr, , sidStr] = activeId.split('-');
                  const rid = parseInt(ridStr);
                  const r = regions.find(r => r.region_id === rid);
                  const s = r?.slices.find(s => s.slice_id === parseInt(sidStr));
                  return (
                    <Chip label={s?.custom_text ?? (s?.translated_content || s?.content || `#${sidStr}`)} size="medium"
                      color="primary" variant="filled" sx={{ fontSize: '0.85rem', py: 0.5, boxShadow: 3 }}
                      title={s?.translated_content ? s.content : undefined} />
                  );
                })() : null}
              </DragOverlay>
            </DndContext>
          )}
        </Paper>

        {/* 区域 2：提示词库 — 搜索框 + 分类标签树 */}
        <Paper sx={{ p: 2, bgcolor: '#fafbfc' }} variant="outlined">
          <Typography variant="subtitle1" sx={{ mb: 1.5, fontWeight: 600 }}>
            提示词库
          </Typography>
          <RegionPanel types={sliceTypes} onSliceClick={handleSliceClick} />
        </Paper>
      </Box>

      {/* 底部预览栏：拼接后的完整 Prompt 文本 */}
      <Paper
        sx={{
          p: 1.5,
          flexShrink: 0,
          borderTop: 1,
          borderColor: 'divider',
          bgcolor: '#f8f9fa',
          fontFamily: 'monospace',
          fontSize: '0.8rem',
          color: 'text.secondary',
        }}
        variant="outlined"
      >
        📝 {preview || '从提示词库中选择标签开始构建...'}
      </Paper>
    </Box>
  );
}
