import React, { useState } from 'react';

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
          className="peer opacity-0 w-0 h-0"
        />
        <span
          className={`absolute cursor-pointer top-0 left-0 right-0 bottom-0 bg-gray-300 transition-all duration-400 rounded-[24px] before:content-[''] before:absolute before:h-[18px] before:w-[18px] before:left-[3px] before:bottom-[3px] before:bg-white before:transition-all before:duration-[400ms] before:rounded-full peer-checked:before:translate-x-[22px] ${
            isOn ? 'bg-blue-500' : ''
          }`}
          style={{ height: '24px', top: '10px' }}
        ></span>
      </label>
      <span>{label}</span>
    </div>
  );
};

export default ToggleSlider;
