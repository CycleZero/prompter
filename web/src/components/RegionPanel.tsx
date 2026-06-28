import { useState } from 'react';
import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Typography,
  CircularProgress,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { SliceType, Slice } from '../types';
import { api } from '../api/client';

// 提示词库面板属性
interface RegionPanelProps {
  types: SliceType[];
  onSliceClick: (typeName: string, slice: Slice) => void;
}

/** 片段类型树节点 — 递归渲染手风琴，展开时懒加载该类型下的片段 */
function TypeNode({
  type,
  level,
  onSliceClick,
}: {
  type: SliceType;
  level: number;
  onSliceClick: (typeName: string, slice: Slice) => void;
}) {
  const [slices, setSlices] = useState<Slice[] | null>(null);
  const [loading, setLoading] = useState(false);

  // 展开时懒加载片段列表
  const handleExpand = () => {
    if (slices === null && !loading) {
      setLoading(true);
      api
        .listSlicesByType(type.id)
        .then((res) => setSlices(res.data.list))
        .catch(() => setSlices([]))
        .finally(() => setLoading(false));
    }
  };

  return (
    <>
      <Accordion
        key={type.id}
        disableGutters
        onChange={(_event, expanded) => {
          if (expanded) handleExpand();
        }}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ ml: level * 2 }}>
          <Typography variant="body2">{type.name}</Typography>
        </AccordionSummary>
        <AccordionDetails
          sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}
        >
          {loading ? (
            <CircularProgress size={20} />
          ) : slices && slices.length === 0 ? (
            <Typography variant="caption" color="text.secondary">
              暂无片段
            </Typography>
          ) : (
            slices?.map((s) => (
              <Chip
                key={s.id}
                label={s.content}
                size="small"
                variant="outlined"
                onClick={() => onSliceClick(type.name, s)}
                sx={{ cursor: 'pointer' }}
              />
            ))
          )}
        </AccordionDetails>
      </Accordion>
      {/* 递归渲染子类型 */}
      {type.children?.map((child) => (
        <TypeNode
          key={child.id}
          type={child}
          level={level + 1}
          onSliceClick={onSliceClick}
        />
      ))}
    </>
  );
}

/** 提示词库面板 — 以 SliceType 树形式展示，点击片段触发选择 Region 的对话框 */
export function RegionPanel({ types, onSliceClick }: RegionPanelProps) {
  if (!types.length) {
    return (
      <Typography variant="body2" color="text.secondary">
        暂无提示词类型数据
      </Typography>
    );
  }

  return (
    <>
      {types.map((t) => (
        <TypeNode
          key={t.id}
          type={t}
          level={0}
          onSliceClick={onSliceClick}
        />
      ))}
    </>
  );
}
