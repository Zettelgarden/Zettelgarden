import React, { useState } from 'react';
import styles from './ToggleSlider.module.css';

interface ToggleSliderProps {
  label: string;
  initialState: boolean;
  onToggle: (toggle: boolean) => void;
}

export const ToggleSlider = ({
  label,
  initialState = false,
  onToggle,
}: ToggleSliderProps) => {
  const [isOn, setIsOn] = useState(initialState);

  const handleToggle = () => {
    setIsOn(!isOn);
    if (onToggle) {
      onToggle(!isOn);
    }
  };

  return (
    <div>
      <label className="relative inline-block w-[46px] h-[44px]">
        <input
          type="checkbox"
          checked={isOn}
          onChange={handleToggle}
          className="opacity-0 w-0 h-0"
        />
        <span
          className={`absolute cursor-pointer top-0 left-0 right-0 bottom-0 bg-gray-300 transition-all duration-400 rounded-[24px] ${
            styles.sliderKnob
          } ${isOn ? 'bg-blue-500 ' + styles.sliderKnobChecked : ''}`}
          style={{ height: '24px', top: '10px' }}
        ></span>
      </label>
      <span>{label}</span>
    </div>
  );
};

export default ToggleSlider;
