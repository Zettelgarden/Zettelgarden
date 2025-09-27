import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Tag } from "../../models/Tags";
import { deleteTag } from "../../api/tags";
import { useTagContext } from "../../contexts/TagContext";
import { PopupMenu } from "../common/PopupMenu";

interface TagListItemInterface {
  tag: Tag;
}

export function TagListItem({ tag }: TagListItemInterface) {
  const [showMenu, setShowMenu] = useState<boolean>(false);
  const { setRefreshTags } = useTagContext();
  const navigate = useNavigate();

  function toggleMenu() {
    setShowMenu(!showMenu);
  }

  function handleViewCards() {
    let searchTerm = "#" + tag.name;
    navigate(`/app/search?term=${encodeURIComponent(searchTerm)}`);
  }

  function handleViewTasks() {
    let searchTerm = "#" + tag.name;
    navigate(`/app/tasks?term=${encodeURIComponent(searchTerm)}`);
  }

  async function handleDelete() {
    let _ = await deleteTag(tag.id)
      .then((data) => {
        setRefreshTags(true);
      })
      .catch((error) =>
        alert("Unable to delete tag.")
      );
    setShowMenu(false);
  }

  const menuOptions = [
    { label: "View Cards", onClick: handleViewCards },
    { label: "View Tasks", onClick: handleViewTasks },
    { label: "Delete", onClick: handleDelete, className: "text-red-600" }
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-4 hover:shadow-lg transition-all relative">
      <div className="flex justify-between items-center mb-2">
        <h3 className="text-lg font-semibold">{tag.name}</h3>
        <button onClick={toggleMenu} className="text-gray-600 hover:text-blue-600 cursor-pointer">
          ⋮
        </button>
      </div>
      <PopupMenu options={menuOptions} isOpen={showMenu} />
    </div>
  );
}
